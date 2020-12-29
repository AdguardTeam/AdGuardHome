interface ErrorCheck<T = any> {
    error?: Error;
    result?: T;
}

export function errorChecker<T = any>(response: Error | any): ErrorCheck<T> {
    if (typeof response !== 'object') {
        return { result: response };
    }
    if (response instanceof Error) {
        return { error: response };
    }
    return { result: response };
}
