import { defineConfig } from 'orval';

export default defineConfig({
    adguardHome: {
        input: {
            target: '../openapi/openapi.yaml',
        },
        output: {
            target: './src/api/generated.ts',
            schemas: './src/api/model',
            client: 'fetch',
            baseUrl: 'control',
            prettier: true,
            override: {
                header: false,
                mutator: {
                    path: './src/api/customFetch.ts',
                    name: 'customFetch',
                },
                fetch: {
                    includeHttpResponseReturnType: false,
                },
                enumGenerationType: 'union',
            },
        },
    },
});
