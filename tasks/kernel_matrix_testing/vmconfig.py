import copy
import itertools
import json
import math
import os
import platform

from .init_kmt import KMT_STACKS_DIR, VMCONFIG, KMT_ROOTFS_DIR, check_and_get_stack
from .stacks import create_stack, stack_exists
from urllib.parse import urlparse
from .tool import Exit, ask, info, warn

vm_recipe = "recipe"
vm_architecture = "arch"
vm_version = "version"
local_arch = "local"

try:
    from thefuzz import fuzz, process
except ImportError:
    process = None
    fuzz = None

kernels = [
    "5.0",
    "5.1",
    "5.2",
    "5.3",
    "5.4",
    "5.5",
    "5.6",
    "5.7",
    "5.8",
    "5.9",
    "5.10",
    "5.11",
    "5.12",
    "5.13",
    "5.14",
    "5.15",
    "5.16",
    "5.17",
    "5.18",
    "5.19",
    "4.4",
    "4.5",
    "4.6",
    "4.7",
    "4.8",
    "4.9",
    "4.10",
    "4.11",
    "4.12",
    "4.13",
    "4.14",
    "4.15",
    "4.16",
    "4.17",
    "4.18",
    "4.19",
    "4.20",
]
distributions = {
    # Ubuntu mappings
    "ubuntu_16": "ubuntu_16.04",
    "ubuntu_18": "ubuntu_18.04",
    "ubuntu_20": "ubuntu_20.04",
    "ubuntu_22": "ubuntu_22.04",
    "ubuntu_23": "ubuntu_23.10",
    "xenial": "ubuntu_16.04",
    "bionic": "ubuntu_18.04",
    "focal": "ubuntu_20.04",
    "jammy": "ubuntu_22.04",
    "mantic": "ubuntu_23.10",
    # Amazon Linux mappings
    "amazon_4.14": "amzn_4.14",
    "amazon_5.4": "amzn_5.4",
    "amazon_5.10": "amzn_5.10",
    "amzn_4.14": "amzn_4.14",
    "amzn_5.4": "amzn_5.4",
    "amzn_5.10": "amzn_5.10",
    # Fedora mappings
    "fedora_37": "fedora_37",
    "fedora_38": "fedora_38",
    # Debian mappings
    "debian_10": "debian_10",
    "debian_11": "debian_11",
    "debian_12": "debian_12",
    # CentOS mappings
    "centos_79": "centos_79",
}

arch_mapping = {"amd64": "x86_64", "x86": "x86_64", "x86_64": "x86_64", "arm64": "arm64", "arm": "arm64", "aarch64": "arm64"}

TICK = "\u2713"
CROSS = "\u2718"
table = [
    ["Image", "x86_64", "arm64"],
    ["ubuntu-18 (bionic)", TICK, CROSS],
    ["ubuntu-20 (focal)", TICK, TICK],
    ["ubuntu-22 (jammy)", TICK, TICK],
    ["amazon linux 2 - v4.14", TICK, TICK],
    ["amazon linux 2 - v5.4", TICK, TICK],
    ["amazon linux 2 - v5.10", TICK, TICK],
    ["amazon linux 2 - v5.15", TICK, CROSS],
    ["fedora 35 - v5.14.10", TICK, TICK],
    ["fedora 36 - v5.17.5", TICK, TICK],
    ["fedora 37 - v6.0.7", TICK, TICK],
    ["fedora 38 - v6.2.9", TICK, TICK],
    ["debian 10 - v4.19.0", TICK, TICK],
    ["debian 11 - v5.10.0", TICK, TICK],
]

consoles = {"x86_64": "ttyS0", "arm64": "ttyAMA0"}

def lte_414(version):
    major, minor = version.split('.')
    return (int(major) <= 4) and (int(minor) <= 14)

def get_image_list(distro, custom):
    custom_kernels = list()
    for k in kernels:
        if lte_414(k):
            custom_kernels.append([f"custom kernel v{k}", TICK, CROSS])
        else:
            custom_kernels.append([f"custom kernel v{k}", TICK, TICK])

    if (not (distro or custom)) or (distro and custom):
        return table + custom_kernels
    if distro:
        return table
    if custom:
        return custom_kernels


def power_log_str(x):
    num = int(x)
    return str(2 ** (math.ceil(math.log(num, 2))))


def mem_to_pow_of_2(memory):
    for i in range(len(memory)):
        new = power_log_str(memory[i])
        if new != memory[i]:
            info(f"rounding up memory: {memory[i]} -> {new}")
            memory[i] = new


def check_memory_and_vcpus(memory, vcpus):
    for mem in memory:
        if not mem.isnumeric() or int(mem) == 0:
            raise Exit(f"Invalid values for memory provided {memory}")

    for v in vcpus:
        if not v.isnumeric or int(v) == 0:
            raise Exit(f"Invalid values for vcpu provided {v}")


def empty_config(file_path):
    j = json.dumps({"vmsets": []}, indent=4)
    with open(file_path, 'w') as f:
        f.write(j)


def list_possible():
    distros = list(distributions.keys())
    archs = list(arch_mapping.keys())
    archs.append(local_arch)

    result = list()
    possible = list(itertools.product(["custom"], kernels, archs)) + list(itertools.product(["distro"], distros, archs))
    for p in possible:
        result.append(f"{p[0]}-{p[1]}-{p[2]}")

    return result

# normalize_vm_def converts the detected user provider vm-def
# to a standard form with consisten values for
# recipe: [custom, distro]
# version: [4.4, 4.5, ..., 5.15, jammy, focal, bionic]
# arch: [x86_64, amd64]
# Each normalized_vm_def output corresponds to each VM
# requested by the user
def normalize_vm_def(possible, vm):
    # atempt to fuzzy match user provided vm-def with the possible list.
    vm_def, _ = process.extractOne(vm, possible, scorer=fuzz.token_sort_ratio)
    recipe, version, arch = vm_def.split('-')

    if arch != local_arch:
        arch = arch_mapping[arch]

    if recipe == "distro":
        version = distributions[version]

    return recipe, version, arch

def get_custom_kernel_config(template, recipe, version, arch):
    if arch == local_arch:
        arch = arch_mapping[platform.machine()]

    if arch == "x86_64":
        console = "ttyS0"
    else:
        console = "ttyAMA0"

    if lte_414(version):
        extra_params = {
            "console": console,
            "systemd.unified_cgroup_hierarchy": "0"
        }
    else:
        extra_params = {
            "console": console,
        }

    return {
        "dir": f"kernel-{version}.{arch}.pkg",
        "tag": version,
        "extra_params": extra_params,
    }

# This function derives the configuration for each
# unique kernel or distribution from the normalized vm-def.
# For more details on the generated configuration element, refer
# to the micro-vms scenario in test-infra-definitions
def get_kernel_config(template, recipe, version, arch):
    if recipe == "custom":
        return get_custom_kernel_config(template, recipe, version, arch)

    if arch == "local":
        arch = arch_mapping[platform.machine()]

    setname = f"{recipe}_{arch}"

    for vmset in template["vmsets"]:
        if vmset["name"] != setname:
            continue

        for kernel in vmset["kernels"]:
            if kernel["tag"] == version:
                return copy.deepcopy(kernel)

    raise Exit(f"No kernel {version} in set {setname}")


def vmset_exists(vm_config, setname):
    vmsets = vm_config["vmsets"]

    for vmset in vmsets:
        if vmset["name"] == setname:
            return True

    return False


def kernel_in_vmset(vmset, kernel):
    vmset_kernels = vmset["kernels"]
    for k in vmset_kernels:
        if k["tag"] == kernel["tag"]:
            return True

    return False

def get_vmconfig_file():
    return "test/new-e2e/system-probe/config/vmconfig.json"

def vmset_name(arch, recipe, setprefix):
    name = f"{recipe}_{arch}"
    if setprefix != "":
        return f"{setprefix}_{name}"

    return name

def add_custom_vmset(vmset, vm_config):
    arch = vmset.arch
    if arch == local_arch:
        arch = arch_mapping[platform.machine()]

    lte = False
    for vm in vmset.vms:
        if lte_414(vm.version):
            lte = True
            break

    image_path = f"custom-bullseye.{arch}.qcow2"
    if lte:
        image_path = f"custom-buster.{arch}.qcow2"

    if vmset_exists(vm_config, vmset.name):
        return

    new_set = {
        "name": vmset.name,
        "recipe": f"{vmset.recipe}-{vmset.arch}",
        "arch": vmset.arch,
        "kernels": list(),
        "image": {
            "image_path": image_path,
            "image_source": f"https://dd-agent-omnibus.s3.amazonaws.com/kernel-version-testing/rootfs/{image_path}"
        }
    }

    vm_config["vmsets"].append(new_set)


def add_vmset(vmset, vm_config):
    if vmset_exists(vm_config, vmset.name):
        return

    if vmset.recipe == "custom":
        return add_custom_vmset(vmset, vm_config)

    new_set = {
        "name": vmset.name,
        "recipe": f"{vmset.recipe}-{vmset.arch}",
        "arch": vmset.arch,
        "kernels": list(),
    }

    vm_config["vmsets"].append(new_set)


def add_kernel(vm_config, kernel, setname):
    for vmset in vm_config["vmsets"]:
        if vmset["name"] != setname:
            continue

        if not kernel_in_vmset(vmset, kernel):
            vmset["kernels"].append(kernel)
            return

    raise Exit(f"Unable to find vmset with name {setname}")

def add_vcpu(vmset, vcpu):
    vmset["vcpu"] = vcpu

def add_memory(vmset, memory):
    vmset["memory"] = memory

def template_name(arch, recipe):
    if arch == local_arch:
        arch = arch_mapping[platform.machine()]

    recipe_without_arch = recipe.split("-")[0]
    return f"{recipe_without_arch}_{arch}"

def add_disks(vmconfig_template, vmset):
    tname = template_name(vmset["arch"], vmset["recipe"])

    for template in vmconfig_template["vmsets"]:
        if template["name"] == tname:
            vmset["disks"] = copy.deepcopy(template["disks"])

def add_console(vmset):
    vmset["console_type"] = "file"

def url_to_fspath(url):
    source = urlparse(url)
    if os.path.basename(source.path).endswith(".xz"):
        filename = os.path.basename(source.path)[:-len(".xz")]
    else:
        filename = os.path.basename(source.path)

    return f"file://{os.path.join(KMT_ROOTFS_DIR,filename)}"

def image_source_to_path(vmset):
    if vmset["recipe"] == f"custom-{vmset['arch']}":
        vmset["image"]["image_source"] = url_to_fspath(vmset["image"]["image_source"])
        return

    for kernel in vmset["kernels"]:
        kernel["image_source"] = url_to_fspath(kernel["image_source"])

    if "disks" in vmset:
        for disk in vmset["disks"]:
            disk["source"] = url_to_fspath(disk["source"])

class VM:
    def __init__(self, version):
        self.version = version

class VMSet:
    def __init__(self, arch, recipe, name):
        self.arch = arch
        self.recipe = recipe
        self.name = name
        self.vms = list()

    def __eq__(self, other):
        return self.name == other.name

    def __hash__(self):
        return hash(self.name)

    def __repr__(self):
        vm_str = list()
        for vm in self.vms:
            vm_str.append(vm.version)
        return f"<VMSet> name={self.name} arch={self.arch} vms={','.join(vm_str)}"

    def add_vm_if_belongs(self, recipe, version, arch):
        if recipe == "custom":
            expected_prefix = custom_version_prefix(version)
            if not self.name.startswith(expected_prefix):
                return

        if self.recipe == recipe and self.arch == arch:
            self.vms.append(VM(version))

def custom_version_prefix(version):
    return "lte_414" if lte_414(version) else "gt_414"

def generate_vmconfig(vm_config, normalized_vm_defs, vcpu, memory, sets, ci):
    with open(get_vmconfig_file()) as f:
        vmconfig_template = json.load(f)

    # generate all vmsets
    vmsets = set()
    for recipe, version, arch in normalized_vm_defs:
        if recipe == "custom":
            sets.append(custom_version_prefix(version))

        # duplicate vm if multiple sets provided by user
        for s in sets:
            vmsets.add(VMSet(arch, recipe, vmset_name(arch, recipe, s)))

        if len(sets) == 0:
            vmsets.add(VMSet(arch, recipe, vmset_name(arch, recipe, "")))

    # map vms to vmsets
    for recipe, version, arch in normalized_vm_defs:
        for vmset in vmsets:
            vmset.add_vm_if_belongs(recipe, version, arch)

    # add new vmsets to new vm_config
    for vmset in vmsets:
        add_vmset(vmset, vm_config)

    # add vm configurations to vmsets.
    for vmset in vmsets:
        for vm in vmset.vms:
            add_kernel(vm_config, get_kernel_config(vmconfig_template, vmset.recipe, vm.version, vmset.arch), vmset.name)

    for vmset in vm_config["vmsets"]:
        add_vcpu(vmset, vcpu)
        add_memory(vmset, memory)

        if vmset["recipe"] != "custom":
            add_disks(vmconfig_template, vmset)

        # For local VMs we want to read images from the filesystem
        if vmset["arch"] == local_arch:
            image_source_to_path(vmset)

        if ci:
            add_console(vmset)

    return vm_config

def ls_to_int(ls):
    int_ls = list()
    for elem in ls:
        int_ls.append(int(elem))

    return int_ls


def gen_config_for_stack(ctx, stack, vms, sets, init_stack, vcpu, memory, new, ci):
    stack = check_and_get_stack(stack)
    if not stack_exists(stack) and not init_stack:
        raise Exit(
            f"Stack {stack} does not exist. Please create stack first 'inv kmt.stack-create --stack={stack}, or specify --init-stack option'"
        )

    if init_stack:
        create_stack(ctx, stack)

    info(f"[+] Select stack {stack}")

    # get normalized vm definitions
    vm_types = vms.split(',')
    if len(vm_types) == 0:
        raise Exit("No VMs to boot provided")

    ## get all possible (recipe, version, arch) combinations we can support.
    possible = list_possible()
    normalized_vms = list()
    for vm in vm_types:
        normalized_vms.append(normalize_vm_def(possible, vm))

    vmconfig_file = f"{KMT_STACKS_DIR}/{stack}/{VMCONFIG}"
    if new or not os.path.exists(vmconfig_file):
        empty_config(vmconfig_file)

    with open(vmconfig_file) as f:
        orig_vm_config = f.read()
    vm_config = json.loads(orig_vm_config)

    vm_config = generate_vmconfig(vm_config, normalized_vms, vcpu, memory, sets, ci)
    vm_config_str = json.dumps(vm_config, indent=4)

    tmpfile = "/tmp/vm.json"
    with open(tmpfile, "w") as f:
        f.write(vm_config_str)

    if new:
        empty_config("/tmp/empty.json")
        ctx.run(f"git diff /tmp/empty.json {tmpfile}", warn=True)
    else:
        ctx.run(f"git diff {vmconfig_file} {tmpfile}", warn=True)

    if ask("are you sure you want to apply the diff? (y/n)") != "y":
        warn("[-] diff not applied")
        return

    with open(vmconfig_file, "w") as f:
        f.write(vm_config_str)

    info(f"[+] vmconfig @ {vmconfig_file}")

def list_all_distro_normalized_vms(archs):
    with open(get_vmconfig_file()) as f:
        vmconfig = json.load(f)

    vms = list()
    for arch in archs:
        distributions = list()
        for vmset in vmconfig["vmsets"]:
            if vmset["arch"] not in arch:
                continue

            for kernel in vmset["kernels"]:
                distributions.append(kernel["tag"])

        for distro_version in distributions:
            vms.append(("distro", distro_version, arch))

    return vms


def gen_config(ctx, stack, vms, sets, init_stack, vcpu, memory, new, ci, arch, output_file):
    vcpu_ls = vcpu.split(',')
    memory_ls = memory.split(',')

    check_memory_and_vcpus(memory_ls, vcpu_ls)
    mem_to_pow_of_2(memory_ls)
    set_ls = list()
    if sets != "":
        set_ls = sets.split(",")

    if not ci:
        return gen_config_for_stack(ctx, stack, vms, set_ls, init_stack, ls_to_int(vcpu_ls), ls_to_int(memory_ls), new, ci)

    arch_ls = ["x86_64", "arm64"]
    if arch != "":
        arch_ls = [arch_mapping[arch]]

    vms_to_generate = list_all_distro_normalized_vms(arch_ls)

    vm_config = generate_vmconfig({"vmsets":[]}, vms_to_generate, ls_to_int(vcpu_ls), ls_to_int(memory_ls), set_ls, ci)

    with open(output_file, "w") as f:
        f.write(json.dumps(vm_config, indent=4))
