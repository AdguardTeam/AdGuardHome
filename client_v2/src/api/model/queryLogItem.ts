import type { DnsAnswer } from './dnsAnswer';
import type { DnsQuestion } from './dnsQuestion';
import type { FilteringReason } from './filteringReason';
import type { QueryLogItemClient } from './queryLogItemClient';
import type { QueryLogItemClientProto } from './queryLogItemClientProto';
import type { ResultRule } from './resultRule';

/**
 * Query log item
 */
export interface QueryLogItem {
    answer?: DnsAnswer[];
    /** Answer from upstream server (optional) */
    original_answer?: DnsAnswer[];
    /** Defines if the response has been served from cache. */
    cached?: boolean;
    /** Upstream URL starting with tcp://, tls://, https://, or with an IP address. */
    upstream?: string;
    /** If true, the response had the Authenticated Data (AD) flag set. */
    answer_dnssec?: boolean;
    /** The client's IP address. */
    client?: string;
    /** The ClientID, if provided in DoH, DoQ, or DoT. */
    client_id?: string;
    client_info?: QueryLogItemClient;
    client_proto?: QueryLogItemClientProto;
    /** The IP network defined by an EDNS Client-Subnet option in the request message if any. */
    ecs?: string;
    elapsedMs?: string;
    question?: DnsQuestion;
    /**
     * In case if there's a rule applied to this DNS request, this is ID of the filter list that the rule belongs to.
     * Deprecated: use `rules[*].filter_list_id` instead.
     * @deprecated
     */
    filterId?: number;
    /**
     * Filtering rule applied to the request (if any).
     * Deprecated: use `rules[*].text` instead.
     * @deprecated
     */
    rule?: string;
    /** Applied rules. */
    rules?: ResultRule[];
    reason?: FilteringReason;
    /** Set if reason=FilteredBlockedService */
    service_name?: string;
    /** DNS response status */
    status?: string;
    /** DNS request processing start time */
    time?: string;
}
