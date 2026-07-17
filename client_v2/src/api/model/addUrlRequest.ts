/**
 * /add_url request data
 */
export interface AddUrlRequest {
    name?: string;
    /** URL or an absolute path to the file containing filtering rules. */
    url?: string;
    whitelist?: boolean;
}
