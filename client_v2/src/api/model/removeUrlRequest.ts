/**
 * /remove_url request data
 */
export interface RemoveUrlRequest {
    /** Previously added URL containing filtering rules */
    url?: string;
    whitelist?: boolean;
}
