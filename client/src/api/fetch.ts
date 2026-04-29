export type RequestConfig = Omit<RequestInit, 'body' | 'headers' | 'method'> & {
    data?: any;
    headers?: HeadersInit;
};

export type FetchResponse = {
    data: any;
    status: number;
};

type FetchError = Error & {
    response?: FetchResponse;
};

const parseResponseData = async (response: Response) => {
    if (response.status === 204 || response.status === 205) {
        return '';
    }

    const text = await response.text();

    if (text === '') {
        return '';
    }

    try {
        return JSON.parse(text);
    } catch (_error) {
        return text;
    }
};

const shouldEncodeJSON = (contentType: string | null, data: any) =>
    typeof data !== 'string' && (!contentType || contentType.toLowerCase().includes('application/json'));

export const fetchRequest = async (url: string, method = 'GET', config: RequestConfig = {}): Promise<FetchResponse> => {
    const { data, headers: requestHeaders, ...requestInit } = config;
    const headers = new Headers(requestHeaders);
    const init: RequestInit = {
        method,
        headers,
        ...requestInit,
    };

    if (method !== 'GET' && method !== 'HEAD' && data !== undefined) {
        if (!headers.has('Content-Type')) {
            headers.set('Content-Type', 'application/json');
        }

        init.body = shouldEncodeJSON(headers.get('Content-Type'), data) ? JSON.stringify(data) : data;
    }

    const response = await fetch(url, init);
    const responseData = await parseResponseData(response);

    if (!response.ok) {
        const error = new Error(`${url} | ${String(responseData)} | ${response.status}`) as FetchError;

        error.response = {
            data: responseData,
            status: response.status,
        };

        throw error;
    }

    return {
        data: responseData,
        status: response.status,
    };
};
