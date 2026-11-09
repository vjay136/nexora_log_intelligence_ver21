%{!?nli_version:%global nli_version 22.4.0}
%{!?nli_release:%global nli_release 0}
Name:           nli-controller-agent
Version:        %{nli_version}
Release:        %{nli_release}%{?dist}
Summary:        Nexora Log Intelligence Ver22 controller and Web console
License:        LicenseRef-Proprietary
URL:            https://github.com/vjay136/nexora_log_intelligence_ver21.git
Source0:        %{name}-%{version}.tar.gz
ExclusiveArch:  x86_64
Requires(pre):  shadow-utils
Requires(post): systemd coreutils sed grep
Requires(preun): systemd
Requires(postun): systemd
Requires:       python3 curl openssl hostname iproute
AutoReqProv:    no
%global debug_package %{nil}
%global _build_id_links none
%global __os_install_post %{nil}

%description
NLI v21 controller providing authenticated enrollment, fleet synchronization,
rollouts, telemetry APIs, and the byte-verified webview_ver21 console.

%prep
%autosetup -n %{name}-%{version}

%build
# The static binary and Web assets are built and verified before rpmbuild.

%install
install -d %{buildroot}%{_bindir} %{buildroot}%{_sbindir} %{buildroot}%{_sysconfdir}/nli
install -d %{buildroot}/usr/lib/systemd/system %{buildroot}%{_datadir}/nli/webview_ver21 %{buildroot}%{_datadir}/nli/supabase
install -d %{buildroot}%{_datadir}/nli/metrics
install -d %{buildroot}%{_docdir}/%{name} %{buildroot}/var/lib/nli-controller
install -m 0755 bin/nli-controller-agent bin/nli %{buildroot}%{_bindir}/
install -m 0755 sbin/nli-controller-setup %{buildroot}%{_sbindir}/nli-controller-setup
install -m 0755 sbin/nli-supabase-setup %{buildroot}%{_sbindir}/nli-supabase-setup
install -m 0755 sbin/nli-admin-reset %{buildroot}%{_sbindir}/nli-admin-reset
install -m 0644 supabase/nli_schema.sql %{buildroot}%{_datadir}/nli/supabase/nli_schema.sql
install -m 0640 configs/controller.json configs/controller.env configs/geoip.env %{buildroot}%{_sysconfdir}/nli/
install -m 0644 systemd/nli-controller-agent.service systemd/nli-controller-bootstrap.service systemd/nli-geoip-update.service systemd/nli-geoip-update.timer %{buildroot}/usr/lib/systemd/system/
cp -a web/. %{buildroot}%{_datadir}/nli/webview_ver21/
find %{buildroot}%{_datadir}/nli/webview_ver21 -type d -exec chmod 0755 {} +
find %{buildroot}%{_datadir}/nli/webview_ver21 -type f -exec chmod 0644 {} +
install -m 0644 README.md %{buildroot}%{_docdir}/%{name}/README.md
install -m 0644 V22_4_RELEASE_NOTES.md All_NLI_features.md V21_R1_RELEASE_NOTES.md V21_R1_TEST_RESULTS.md V21_R2_RELEASE_NOTES.md V21_R2_TEST_RESULTS.md V21_R3_RELEASE_NOTES.md V21_R3_TEST_RESULTS.md V21_R4_RELEASE_NOTES.md V21_R4_TEST_RESULTS.md V21_R4_SURGICAL_DIAGNOSTICS_AND_VERIFICATION.md V21_R5_RELEASE_NOTES.md V21_R5_TEST_RESULTS.md V21_R6_RELEASE_NOTES.md V21_R6_TEST_RESULTS.md V21_R7_RELEASE_NOTES.md V21_R7_TEST_RESULTS.md V21_R7_MAPPING_DESIGN.md V21_R7_FIELD_MAPPING_GUIDE.md V21_R8_RELEASE_NOTES.md V21_R8_TEST_RESULTS.md V21_R8_DESIGN_ENGINE_DESIGN.md V21_R8_DESIGN_ENGINE_GUIDE.md V21_R8_OUTPUT_ACCOUNTING_GUIDE.md V21_R8_AI_PROVIDER_GUIDE.md V21_R9_RELEASE_NOTES.md V21_R9_TEST_RESULTS.md V21_R9_FIELD_MAPPING_GUIDE.md V21_R10_RELEASE_NOTES.md V21_R10_TEST_RESULTS.md V21_R10_MANAGER_DEMO.md nli-output-audit.py VER21_IMPLEMENTATION_PLAN.md %{buildroot}%{_docdir}/%{name}/
install -m 0644 metrics/README.md metrics/metric-catalog.json metrics/prometheus-scrape.yml metrics/grafana-dashboard.json %{buildroot}%{_datadir}/nli/metrics/

%pre
getent group nli >/dev/null || groupadd -r nli
getent passwd nli >/dev/null || useradd -r -g nli -d /var/lib/nli-controller -s /sbin/nologin nli

%post
config_file=%{_sysconfdir}/nli/controller.json
if grep -Eq '"web_dir"[[:space:]]*:[[:space:]]*"/usr/share/nli/webview_ver(1|5|6|7|8|20)"' "$config_file"; then
  sed -i -E 's#/usr/share/nli/webview_ver(1|5|6|7|8|20)(["/])#/usr/share/nli/webview_ver21\2#g' "$config_file"
fi
secret_file=%{_sysconfdir}/nli/controller.env
chown root:nli "$secret_file"
chmod 0640 "$secret_file"
systemctl daemon-reload >/dev/null 2>&1 || :
systemctl enable nli-controller-bootstrap.service >/dev/null 2>&1 || :
# The GeoIP timer remains disabled until an administrator accepts the upstream
# license in /etc/nli/geoip.env and deliberately enables it.
if [ -s /etc/nli/controller-bootstrap.env ]; then systemctl start nli-controller-bootstrap.service >/dev/null 2>&1 || :; fi

%preun
if [ "$1" -eq 0 ]; then
  systemctl disable --now nli-controller-bootstrap.service nli-controller-agent.service >/dev/null 2>&1 || :
fi

%postun
systemctl daemon-reload >/dev/null 2>&1 || :

%files
%attr(0755,root,root) %{_bindir}/nli-controller-agent
%attr(0755,root,root) %{_bindir}/nli
%attr(0755,root,root) %{_sbindir}/nli-controller-setup
%attr(0755,root,root) %{_sbindir}/nli-supabase-setup
%attr(0755,root,root) %{_sbindir}/nli-admin-reset
%dir %attr(0750,root,nli) %{_sysconfdir}/nli
%config(noreplace) %attr(0640,root,nli) %{_sysconfdir}/nli/controller.json
%config(noreplace) %attr(0640,root,nli) %{_sysconfdir}/nli/controller.env
%config(noreplace) %attr(0640,root,nli) %{_sysconfdir}/nli/geoip.env
%ghost %attr(0600,root,root) %{_sysconfdir}/nli/controller-bootstrap.env
%attr(0644,root,root) /usr/lib/systemd/system/nli-controller-agent.service
%attr(0644,root,root) /usr/lib/systemd/system/nli-controller-bootstrap.service
%attr(0644,root,root) /usr/lib/systemd/system/nli-geoip-update.service
%attr(0644,root,root) /usr/lib/systemd/system/nli-geoip-update.timer
%dir %attr(0755,root,root) %{_datadir}/nli
%dir %attr(0755,root,root) %{_datadir}/nli/webview_ver21
%attr(0644,root,root) %{_datadir}/nli/webview_ver21/*
%dir %attr(0755,root,root) %{_datadir}/nli/supabase
%attr(0644,root,root) %{_datadir}/nli/supabase/nli_schema.sql
%dir %attr(0755,root,root) %{_datadir}/nli/metrics
%attr(0644,root,root) %{_datadir}/nli/metrics/README.md
%attr(0644,root,root) %{_datadir}/nli/metrics/metric-catalog.json
%attr(0644,root,root) %{_datadir}/nli/metrics/prometheus-scrape.yml
%attr(0644,root,root) %{_datadir}/nli/metrics/grafana-dashboard.json
%dir %attr(0750,nli,nli) /var/lib/nli-controller
%doc %{_docdir}/%{name}/README.md
%doc %{_docdir}/%{name}/All_NLI_features.md
%doc %{_docdir}/%{name}/V21_R1_RELEASE_NOTES.md
%doc %{_docdir}/%{name}/V21_R1_TEST_RESULTS.md
%doc %{_docdir}/%{name}/V21_R2_RELEASE_NOTES.md
%doc %{_docdir}/%{name}/V21_R2_TEST_RESULTS.md
%doc %{_docdir}/%{name}/V21_R3_RELEASE_NOTES.md
%doc %{_docdir}/%{name}/V21_R3_TEST_RESULTS.md
%doc %{_docdir}/%{name}/V21_R4_RELEASE_NOTES.md
%doc %{_docdir}/%{name}/V21_R4_TEST_RESULTS.md
%doc %{_docdir}/%{name}/V21_R4_SURGICAL_DIAGNOSTICS_AND_VERIFICATION.md
%doc %{_docdir}/%{name}/V21_R5_RELEASE_NOTES.md
%doc %{_docdir}/%{name}/V21_R5_TEST_RESULTS.md
%doc %{_docdir}/%{name}/V21_R6_RELEASE_NOTES.md
%doc %{_docdir}/%{name}/V21_R6_TEST_RESULTS.md
%doc %{_docdir}/%{name}/V21_R7_RELEASE_NOTES.md
%doc %{_docdir}/%{name}/V21_R7_TEST_RESULTS.md
%doc %{_docdir}/%{name}/V21_R7_MAPPING_DESIGN.md
%doc %{_docdir}/%{name}/V21_R7_FIELD_MAPPING_GUIDE.md
%doc %{_docdir}/%{name}/V21_R8_RELEASE_NOTES.md
%doc %{_docdir}/%{name}/V21_R8_TEST_RESULTS.md
%doc %{_docdir}/%{name}/V21_R8_DESIGN_ENGINE_DESIGN.md
%doc %{_docdir}/%{name}/V21_R8_DESIGN_ENGINE_GUIDE.md
%doc %{_docdir}/%{name}/V21_R8_OUTPUT_ACCOUNTING_GUIDE.md
%doc %{_docdir}/%{name}/V21_R8_AI_PROVIDER_GUIDE.md
%doc %{_docdir}/%{name}/V21_R9_RELEASE_NOTES.md
%doc %{_docdir}/%{name}/V21_R9_TEST_RESULTS.md
%doc %{_docdir}/%{name}/V21_R9_FIELD_MAPPING_GUIDE.md
%doc %{_docdir}/%{name}/V21_R10_RELEASE_NOTES.md
%doc %{_docdir}/%{name}/V21_R10_TEST_RESULTS.md
%doc %{_docdir}/%{name}/V21_R10_MANAGER_DEMO.md
%doc %{_docdir}/%{name}/nli-output-audit.py
%doc %{_docdir}/%{name}/VER21_IMPLEMENTATION_PLAN.md
%doc %{_docdir}/%{name}/V22_4_RELEASE_NOTES.md

%changelog
* Mon Nov 09 2026 Nexora Engineering <engineering@nexora.local> - 22.4.0-0
- Improve parser coverage, hourly output files and replay reporting.

* Sun Nov 08 2026 Nexora Engineering <engineering@nexora.local> - 22.2.0-0
- Add literal mapping controls and key/value input preparation.

* Sun Nov 08 2026 Nexora Engineering <engineering@nexora.local> - 22.1.0-0
- Qualify Ver22.1 runtime and fleet operations.

* Sat Nov 07 2026 Nexora Engineering <engineering@nexora.local> - 21.0.0-10
- Add KV Intelligence Engine authoring and release evidence.

* Fri Nov 06 2026 Nexora Engineering <engineering@nexora.local> - 21.0.0-9
- Preserve structured tail evidence; add dynamic KV EOL mappings and synchronized parser lifecycle.

* Fri Nov 06 2026 Nexora Engineering <engineering@nexora.local> - 21.0.0-8
- Add adaptive UI design systems and preserve the qualified R7 parser baseline.

* Fri Nov 06 2026 Nexora Engineering <engineering@nexora.local> - 21.0.0-7
- Add native composable field mapping and shared preview/runtime mapping documentation.

* Fri Nov 06 2026 Nexora Engineering <engineering@nexora.local> - 21.0.0-6
- Publish role-based administrator review and durable quarantine recovery fixes.

* Wed Nov 04 2026 Nexora Engineering <engineering@nexora.local> - 21.0.0-5
- Publish QRW Steps 3-4 mapping, extraction, and lifecycle visibility documentation.

* Tue Nov 03 2026 Nexora Engineering <engineering@nexora.local> - 21.0.0-4
- Publish Ver21 Release 4 lifecycle plumbing, evidence, and surgical diagnostics documentation.

* Tue Nov 03 2026 Nexora Engineering <engineering@nexora.local> - 21.0.0-3
- Publish Ver21 Release 3 artifacts and migrate compatible legacy WebView roots to webview_ver21.

* Mon Nov 02 2026 Nexora Engineering <engineering@nexora.local> - 21.0.0-1
- Promote the governed parser runtime to Ver21 while preserving RPM identity and in-place upgrade paths.

* Sun Nov 01 2026 Nexora Engineering <engineering@nexora.local> - 20.0.0-6
- Preserve the R4 runtime baseline and correct canonical parser bundle hash approval handling; add R6 operator/test evidence.

* Sun Nov 01 2026 Nexora Engineering <engineering@nexora.local> - 20.0.0-4
- Complete exact candidate preview, governed emergency recovery, parser compatibility, and notification state.

* Fri Oct 30 2026 Nexora Engineering <engineering@nexora.local> - 20.0.0-3
- Reconcile legacy parsers, clear terminal recovery badges, and guide evidence-backed quarantine repair.

* Thu Oct 29 2026 Nexora Engineering <engineering@nexora.local> - 20.0.0-2
- Harden parser catalog recovery and complete the guided rollout workflow.

* Thu Oct 29 2026 Nexora Engineering <engineering@nexora.local> - 20.0.0-1
- Fix exact parser-bundle approval, guided quarantine recovery, live source progress, and resilient TLS synchronization.

* Wed Oct 28 2026 Nexora Engineering <engineering@nexora.local> - 19.0.0-8
- Previous production baseline retained for upgrade and rollback acceptance.
