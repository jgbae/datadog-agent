#undef _WIN32_WINNT
#define _WIN32_WINNT _WIN32_WINNT_WINBLUE // Windows 8.1

#include <windows.h>
#include <evntcons.h>
#include <tdh.h>
#include <inttypes.h>

#define EVENT_FILTER_TYPE_EVENT_ID          (0x80000200)
#define EVENT_FILTER_TYPE_PID               (0x80000004)

// This constant defines the maximum number of filter types supported.
#define MAX_FILTER_SUPPORTED                2

extern void etwCallbackC(PEVENT_RECORD);

static void WINAPI RecordEventCallback(PEVENT_RECORD event)
{
    etwCallbackC(event);
}

static TRACEHANDLE DDStartTracing(LPWSTR name, uintptr_t context)
{
    EVENT_TRACE_LOGFILEW trace = {0};
    trace.LoggerName = name;
    trace.Context = (void*)context;
    trace.ProcessTraceMode = PROCESS_TRACE_MODE_REAL_TIME | PROCESS_TRACE_MODE_EVENT_RECORD;
    trace.EventRecordCallback = RecordEventCallback;

    return OpenTraceW(&trace);
}

static ULONG DDEnableTrace(
    TRACEHANDLE TraceHandle,
    LPCGUID     ProviderId,
    ULONG       ControlCode,
    UCHAR       Level,
    ULONGLONG   MatchAnyKeyword,
    ULONGLONG   MatchAllKeyword,
    ULONG       Timeout,
    ULONGLONG*  PIDs,
    ULONG       PIDCount
)
{
    EVENT_FILTER_DESCRIPTOR eventFilterDescriptors[MAX_FILTER_SUPPORTED];

    ENABLE_TRACE_PARAMETERS enableParameters = { 0 };
    enableParameters.Version = ENABLE_TRACE_PARAMETERS_VERSION_2;
    enableParameters.EnableFilterDesc = &eventFilterDescriptors[0];
    enableParameters.FilterDescCount = 0;

    if (PIDCount > 0)
    {
        eventFilterDescriptors[0].Ptr  = (ULONGLONG)PIDs;
        eventFilterDescriptors[0].Size = (ULONG)(sizeof(PIDCount) * PIDCount);
        eventFilterDescriptors[0].Type = EVENT_FILTER_TYPE_PID;

        enableParameters.FilterDescCount++;
    }

    return EnableTraceEx2(
        TraceHandle,
        ProviderId,
        ControlCode,
        Level,
        MatchAnyKeyword,
        MatchAllKeyword,
        Timeout,
        &enableParameters
    );
}
