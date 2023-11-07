using System;
using Microsoft.Deployment.WindowsInstaller;
using System.Collections.Generic;
using System.Reflection;
using Datadog.CustomActions.Extensions;
using Datadog.CustomActions.Interfaces;

namespace Datadog.CustomActions
{
    public class Telemetry
    {
        private readonly IInstallerHttpClient _client;
        private readonly ISession _session;

        public const string DefaultSite = "datadoghq.com";

        private const string DefaultOrigin = "WindowsAgentInstaller";

        public Telemetry(
            IInstallerHttpClient client,
            ISession session)
        {
            _client = client;
            _session = session;
        }

        public static Telemetry DefaultTelemetry(ISession session)
        {
            return new Telemetry(new InstallerWebClient(), session);
        }

        public void ReportTelemetry(string eventName)
        {
            var apikey = _session.Property("APIKEY");
            if (string.IsNullOrEmpty(apikey))
            {
                _session.Log("API key empty, not reporting telemetry");
                return;
            }
            var site = _session.Property("SITE");
            if (string.IsNullOrEmpty(site))
            {
                site = DefaultSite;
            }
            var uri = $"https://instrumentation-telemetry-intake.{site}/api/v2/apmtelemetry";

            var installerVersion = Assembly.GetExecutingAssembly().GetName().Version;
            var agentVersion = CiInfo.PackageVersion;
            var installId = GetInstallId();
            var origin = GetOrigin();
            var payload = @$"
{{
    ""request_type"": ""apm-onboarding-event"",
    ""api_version"": ""v1"",
    ""payload"": {{
        ""event_name"": ""{eventName}"",
        ""tags"": {{
            ""agent_platform"": ""windows"",
            ""agent_version"": ""{agentVersion}"",
            ""script_version"": ""{installerVersion}"",
            ""install_id"": ""{installId}"",
            ""origin"": ""{origin}""
        }}
    }}
}}";

            _client.Post(uri, payload, new Dictionary<string, string>
            {
                { "DD-Api-Key", apikey },
                { "Content-Type", "application/json" },
            });
        }

        private static ActionResult Report(ISession session, string eventName)
        {
            try
            {
                session.Log("Sending installation telemetry");
                DefaultTelemetry(session).ReportTelemetry(eventName);
            }
            catch (Exception e)
            {
                // No need for full stack trace here.
                session.Log($"Error sending telemetry: {e.Message}");
                return ActionResult.Failure;
            }
            return ActionResult.Success;
        }

        private string GetInstallId() {
            var installId = System.Environment.GetEnvironmentVariable("DD_INSTALL_ID");
            if (installId != null && installId.Length > 0) {
                return installId;
            }
            installId = System.Guid.NewGuid().ToString();
            System.Environment.SetEnvironmentVariable("DD_INSTALL_ID", installId);
            return installId;
        }

        private string GetOrigin() {
            var origin = System.Environment.GetEnvironmentVariable("DD_ORIGIN");
            if (origin != null && origin.Length > 0) {
                return origin;
            }
            return DefaultOrigin;
        }

        [CustomAction]
        public static ActionResult ReportFailure(Session session)
        {
            return Report(new SessionWrapper(session), "agent.installation.error");
        }

        [CustomAction]
        public static ActionResult ReportSuccess(Session session)
        {
            return Report(new SessionWrapper(session), "agent.installation.success");
        }
    }
}
