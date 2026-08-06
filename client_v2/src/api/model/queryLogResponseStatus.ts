export type QueryLogResponseStatus =
    | 'all'
    | 'filtered'
    | 'blocked'
    | 'blocked_safebrowsing'
    | 'blocked_parental'
    | 'whitelisted'
    | 'rewritten'
    | 'safe_search'
    | 'processed';
