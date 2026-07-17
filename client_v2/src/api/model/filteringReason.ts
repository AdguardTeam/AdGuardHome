/**
 * Request filtering status.
 */
export type FilteringReason =
    | 'NotFilteredNotFound'
    | 'NotFilteredWhiteList'
    | 'NotFilteredError'
    | 'FilteredBlackList'
    | 'FilteredSafeBrowsing'
    | 'FilteredParental'
    | 'FilteredInvalid'
    | 'FilteredSafeSearch'
    | 'FilteredBlockedService'
    | 'Rewrite'
    | 'RewriteEtcHosts'
    | 'RewriteRule';
