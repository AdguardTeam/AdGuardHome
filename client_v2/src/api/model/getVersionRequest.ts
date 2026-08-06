/**
 * /version.json request data
 */
export interface GetVersionRequest {
    /** If false, server will check for a new version data only once in several hours. */
    recheck_now?: boolean;
}
