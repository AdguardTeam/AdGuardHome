export interface QueryLogAnswer {
  type?: string;
  ttl?: number;
  value?: string;
}

export interface QueryLogRecord {
  client?: string;
  client_info?: {
    name?: string;
    whois?: Record<string, unknown>;
    disallowed?: boolean;
    disallowed_rule?: string;
  };
  qhost?: string;
  time?: string;
  upstream?: string;
  status?: string;
  reason?: string;
  rule?: string;
  question?: {
    host?: string;
    name?: string;
    type?: string;
  };
  answer?: QueryLogAnswer[];
  original_answer?: QueryLogAnswer[];
}
