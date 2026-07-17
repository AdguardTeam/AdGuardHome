import { HTML_PAGES, R_PATH_LAST_PART } from '../helpers/constants';

type CustomFetchOptions = RequestInit & {
    skipAuthRedirect?: boolean;
};

/**
 * Custom fetch mutator for orval-generated API client.
 *
 * Custom fetch wrapper providing AdGuard Home-specific behaviors:
 * - Sets `Content-Type: application/json` on requests with a body
 * - Parses response as JSON (falls back to raw text if JSON.parse fails)
 * - On 403 and not on login/install page → redirects browser to login
 *   and returns `false` (does not throw)
 * - On 403 while on login/install page → throws Error normally
 * - Throws Error with format `${path} | ${data} | ${status}` for all
 *   other non-ok responses
 * - On empty response body → returns empty string ''
 * - On 204 → returns empty string ''
 */
export const customFetch = async <T>(url: string, options?: CustomFetchOptions): Promise<T> => {
    const { skipAuthRedirect, ...fetchOptions } = options || {};

    const fullUrl = url;
    const headers: Record<string, string> = {};

    // Preserve any headers passed in options
    if (fetchOptions.headers) {
        const incomingHeaders = fetchOptions.headers as Record<string, string>;
        Object.assign(headers, incomingHeaders);
    }

    try {
        const response = await fetch(fullUrl, {
            ...fetchOptions,
            headers,
        });

        const text = await response.text();
        const data: T = text
            ? (() => {
                  try {
                      return JSON.parse(text);
                  } catch {
                      return text;
                  }
              })()
            : ('' as unknown as T);

        if (!response.ok) {
            const { pathname } = document.location;
            const shouldRedirect = pathname !== HTML_PAGES.LOGIN && pathname !== HTML_PAGES.INSTALL;

            if (response.status === 403 && shouldRedirect && !skipAuthRedirect) {
                const loginPageUrl = window.location.href.replace(
                    R_PATH_LAST_PART,
                    HTML_PAGES.LOGIN,
                );
                window.location.replace(loginPageUrl);
                return false as unknown as T;
            }

            throw new Error(
                `${fullUrl} | ${typeof data === 'string' ? data : JSON.stringify(data)} | ${response.status}`,
            );
        }

        return data;
    } catch (error) {
        if (error instanceof Error && error.message.includes('|')) {
            throw error;
        }

        throw new Error(`${fullUrl} | ${error instanceof Error ? error.message : String(error)}`);
    }
};

export default customFetch;
