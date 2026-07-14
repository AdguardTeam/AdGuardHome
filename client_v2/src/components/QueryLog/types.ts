export type ResponseEntry = {
    value: string;
    type?: string;
    ttl?: number;
};

export type WhoisInfo = {
    city?: string;
    country?: string;
    descr?: string;
    netname?: string;
    orgname?: string;
};

export type RuleInfo = {
    filter_list_id: number;
    text: string;
};

export type ClientInfo = {
    name?: string;
    ids?: string[];
    tags?: string[];
    disallowed?: boolean;
    disallowed_rule?: string;
    whois?: WhoisInfo;
};

export type TrackerInfo = {
    name: string;
    category: string;
    url?: string;
    sourceData?: {
        name: string;
        url: string;
    } | null;
};

export type Service = {
    id: string;
    name: string;
};

export type LogEntry = {
    time: string;
    domain: string;
    unicodeName?: string;
    type: string;
    response: ResponseEntry[];
    reason: string;
    client: string;
    client_info: ClientInfo | null;
    tracker: TrackerInfo | null;
    upstream: string;
    elapsedMs: string;
    originalResponse: ResponseEntry[];
    status: string;
    service_name: string;
    serviceName: string;
    filterId: number;
    rule: string;
    rules: RuleInfo[];
    answer_dnssec: boolean;
    client_proto: string;
    client_id: string;
    ecs: string;
    cached: boolean;
};
