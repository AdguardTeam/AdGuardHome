import {
  updateQueryLogConfig,
  type QueryLogConfig,
} from '../shared/adguard/querylog.ts';

export async function setQueryLogSettings(options: {
  baseUrl: string;
  config: QueryLogConfig;
  headers?: Record<string, string>;
}) {
  await updateQueryLogConfig({
    baseUrl: options.baseUrl,
    headers: options.headers,
  }, options.config);
}
