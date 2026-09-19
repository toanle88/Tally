#!/usr/bin/env bash
set -Eeuo pipefail

script="$(cd "$(dirname "$BASH_SOURCE")" && pwd)/azure-learning-smoke.sh"

bash "$script" --self-test >/dev/null

if ENVIRONMENT=prod-reference bash "$script" run >/tmp/tally-smoke-invalid.out 2>&1; then
	echo 'smoke runner accepted an unsupported environment' >&2
	exit 1
fi
grep -q 'ENVIRONMENT must be dev or demo' /tmp/tally-smoke-invalid.out

if ENVIRONMENT=dev bash "$script" run >/tmp/tally-smoke-missing.out 2>&1; then
	echo 'smoke runner accepted missing backend credentials' >&2
	exit 1
fi
grep -q 'ARM_SUBSCRIPTION_ID is required' /tmp/tally-smoke-missing.out

for token in PostgreSQLLogs AllMetrics log-analytics AcrPull 'Key Vault Secrets User' '/health/live'; do
	grep -q "$token" "$script"
done

if command -v node >/dev/null 2>&1; then
	test_root="$(mktemp -d)"
	fake_bin="$test_root/bin"
	mkdir -p "$fake_bin"
	digest='sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
	cat > "$test_root/outputs.json" <<EOF
{"environment_profile":{"value":"dev"},"resource_group_id":{"value":"/fake/rg"},"container_app_environment_id":{"value":"/fake/env"},"api_id":{"value":"/fake/api"},"static_web_app_id":{"value":"/fake/swa"},"api_url":{"value":"https://api.example"},"worker_id":{"value":"/fake/worker"},"postgres_server_id":{"value":"/fake/postgres"},"database_id":{"value":"/fake/db"},"key_vault_id":{"value":"/fake/kv"},"log_analytics_workspace_id":{"value":"/fake/la"},"application_insights_id":{"value":"/fake/ai"},"api_identity_id":{"value":"/fake/api-identity"},"api_identity_principal_id":{"value":"api-principal"},"worker_identity_id":{"value":"/fake/worker-identity"},"worker_identity_principal_id":{"value":"worker-principal"},"container_registry_id":{"value":"/fake/acr"},"container_registry_name":{"value":"fakeacr"},"api_image_repository":{"value":"api"},"api_image_digest":{"value":"$digest"},"worker_image_repository":{"value":"worker"},"worker_image_digest":{"value":"$digest"}}
EOF
	cat > "$fake_bin/terraform" <<'EOF'
#!/usr/bin/env bash
if [[ "$*" == *" output "* ]]; then cat "$SMOKE_TEST_OUTPUTS"; fi
EOF
	cat > "$fake_bin/az" <<'EOF'
#!/usr/bin/env bash
case "$1 $2 $3" in
  "account show --subscription") echo "$ARM_SUBSCRIPTION_ID" ;;
  "resource show --ids") echo '{"id":"/fake/resource","properties":{"provisioningState":"Succeeded"}}' ;;
  "containerapp show --ids") identity='/fake/api-identity'; [[ "$*" == *"/fake/worker"* ]] && identity='/fake/worker-identity'; echo "{\"properties\":{\"provisioningState\":\"Succeeded\",\"template\":{\"containers\":[{\"image\":\"fake@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"probes\":[{\"httpGet\":{\"path\":\"/health/live\"}}]}]},\"configuration\":{\"registries\":[{\"identity\":\"$identity\"}]}}}" ;;
  "containerapp revision list") echo '[{"properties":{"runningState":"Running","template":{"containers":[{"image":"fake@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}}}]' ;;
  "acr repository show-manifests") echo 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa' ;;
  "role assignment list") echo '[{"roleDefinitionName":"AcrPull"},{"roleDefinitionName":"Key Vault Secrets User"}]' ;;
  "monitor diagnostic-settings list") echo '{"value":[{"properties":{"logs":[{"category":"PostgreSQLLogs","enabled":true}],"metrics":[{"category":"AllMetrics","enabled":true}]}}]}' ;;
  "containerapp env show") echo 'log-analytics' ;;
  *) exit 1 ;;
esac
EOF
	cat > "$fake_bin/curl" <<'EOF'
#!/usr/bin/env bash
printf '200 https://api.example/health/live 0\n'
EOF
	chmod +x "$fake_bin"/*
	if ! SMOKE_TEST_OUTPUTS="$test_root/outputs.json" SMOKE_REPORT_FILE="$test_root/report.json" ENVIRONMENT=dev ARM_SUBSCRIPTION_ID=sub ARM_TENANT_ID=tenant TF_STATE_RESOURCE_GROUP=state-rg TF_STATE_STORAGE_ACCOUNT=statestore PATH="$fake_bin:$PATH" bash "$script" run > "$test_root/run.out" 2>&1; then
		grep -q 'Smoke report:' "$test_root/run.out"
	else
		echo 'smoke runner did not block absent migration evidence' >&2
		exit 1
	fi
	test -s "$test_root/report.json"
	grep -q '"story_id": "DLV-IAC-002-us8"' "$test_root/report.json"
	grep -q '"cleanup_status": "not-run"' "$test_root/report.json"
	grep -q '"migration_status": "not-verified"' "$test_root/report.json"
	rm -rf "$test_root"
fi

for token in api_id static_web_app_id container_app_environment_id postgres_server_id key_vault_id log_analytics_workspace_id application_insights_id api_identity_id worker_identity_id api_image_repository api_image_digest worker_image_repository worker_image_digest; do
	grep -q "$token" "$script"
done
for token in story_id commit environment started_at completed_at overall_outcome cleanup_status migration_status resource_versions checks limitations; do
	grep -q "$token" "$script"
done
grep -q 'owner' "$script"
grep -q 'dependency' "$script"

! grep -Eq -- '(^|[[:space:]])(-k|--insecure)([[:space:]]|$)' "$script"
! grep -Eq 'terraform[[:space:]]+-chdir=[^[:space:]]+[[:space:]]+(plan|apply|destroy)' "$script"
! grep -Eq 'az[[:space:]]+(resource|containerapp|acr|role|monitor)[[:space:]]+(create|update|delete)' "$script"
! grep -Eq '(migrate\.sh|seed\.sh)' "$script"

echo 'Azure learning smoke runner tests passed'
