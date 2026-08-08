import { defineConfig } from '@hey-api/openapi-ts'

// The generation wrapper overrides `input` with a temporary Redocly bundle.
// Use `make api-ts-generate` as the supported entry point.
export default defineConfig({
  input: 'contracts/openapi/dist/openapi.bundle.yaml',
  output: 'web/src/generated/api',
  plugins: [
    '@hey-api/client-fetch',
    '@hey-api/sdk',
    '@hey-api/typescript',
  ],
})
