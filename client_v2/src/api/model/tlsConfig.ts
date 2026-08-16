import type { TlsConfigKeyType } from './tlsConfigKeyType';

/**
 * TLS configuration settings and status
 */
export interface TlsConfig {
    /** enabled is the encryption (DoT/DoH/HTTPS) status */
    enabled?: boolean;
    /** server_name is the hostname of your HTTPS/TLS server */
    server_name?: string;
    /** if true, forces HTTP->HTTPS redirect */
    force_https?: boolean;
    /** HTTPS port. If 0, HTTPS will be disabled. */
    port_https?: number;
    /** DNS-over-TLS port. If 0, DoT will be disabled. */
    port_dns_over_tls?: number;
    /** DNS-over-QUIC port. If 0, DoQ will be disabled. */
    port_dns_over_quic?: number;
    /** Base64 string with PEM-encoded certificates chain */
    certificate_chain?: string;
    /** Base64 string with PEM-encoded private key */
    private_key?: string;
    /** Set to true if the user has previously saved a private key as a string.  This is used so that the server and the client don't have to send the private key between each other every time, which might lead to security issues. */
    private_key_saved?: boolean;
    /** Path to certificate file */
    certificate_path?: string;
    /** Path to private key file */
    private_key_path?: string;
    /** Set to true if the specified certificates chain is a valid chain of X509 certificates. */
    valid_cert?: boolean;
    /** Set to true if the specified certificates chain is verified and issued by a known CA. */
    valid_chain?: boolean;
    /** The subject of the first certificate in the chain. */
    subject?: string;
    /** The issuer of the first certificate in the chain. */
    issuer?: string;
    /** The NotBefore field of the first certificate in the chain. */
    not_before?: string;
    /** The NotAfter field of the first certificate in the chain. */
    not_after?: string;
    /** The value of SubjectAltNames field of the first certificate in the chain. */
    dns_names?: string[];
    /** Set to true if the key is a valid private key. */
    valid_key?: boolean;
    /** Key type. */
    key_type?: TlsConfigKeyType;
    /** A validation warning message with the issue description. */
    warning_validation?: string;
    /** Set to true if both certificate and private key are correct. */
    valid_pair?: boolean;
    /** Set to true if plain DNS is allowed for incoming requests. */
    serve_plain_dns?: boolean;
    /** DNS-over-HTTPS port. If 0, DNSCrypt will be disabled. */
    port_dnscrypt?: number;
    /** Path to the DNSCrypt configuration file. */
    dnscrypt_config_file?: string;
}
