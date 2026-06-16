import { DNS_RECORD_TYPES } from 'panel/helpers/constants';

export type UserRulesFormValues = {
    userRules: string;
};

export type CheckFormValues = {
    hostname: string;
    client: string;
    qtype: string;
};

export type CheckResultRule = {
    filter_list_id?: number;
    text: string;
};

export type CheckResultData = {
    hostname?: string;
    reason?: string;
    rules?: CheckResultRule[];
    service_name?: string;
    cname?: string;
    ip_addrs?: string[];
};

export type ResultActionKind =
    | 'allow'
    | 'block'
    | 'disable-parental'
    | 'disable-safebrowsing'
    | 'disable-safesearch'
    | 'disable-blocked-service'
    | 'disable-filter'
    | 'edit-rewrite'
    | 'delete-rewrite'
    | 'remove-rewrite-rule'
    | 'none';

export type ResultAction = {
    kind: ResultActionKind;
    label: string;
};

export type RewriteEntry = {
    domain: string;
    answer: string;
    enabled: boolean;
};

export type RewriteDialogState = {
    visible: boolean;
    target: RewriteEntry;
};

export type DnsRecordTypeOption = {
    label: string;
    value: string;
};

export const DNS_RECORD_TYPE_OPTIONS: DnsRecordTypeOption[] = DNS_RECORD_TYPES.map((value) => ({
    label: value,
    value,
}));
