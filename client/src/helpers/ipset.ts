/**
 * IPSet utilities for parsing and validating ipset rules
 * Rule format: DOMAIN[,DOMAIN,...]/IPSET_NAME[,IPSET_NAME,...]
 */

export interface IPSetRule {
    domains: string[];
    ipsets: string[];
}

/**
 * Parse ipset rule string into domains and ipsets
 */
export const parseIPSetRule = (rule: string): IPSetRule | null => {
    const trimmedRule = rule.trim();

    if (!trimmedRule) {
        return null;
    }

    const separatorIndex = trimmedRule.indexOf('/');

    if (separatorIndex === -1) {
        return null;
    }

    const domainsPart = trimmedRule.substring(0, separatorIndex);
    const ipsetsPart = trimmedRule.substring(separatorIndex + 1);

    const domains = domainsPart
        .split(',')
        .map((d) => d.trim())
        .filter((d) => d.length > 0);

    const ipsets = ipsetsPart
        .split(',')
        .map((s) => s.trim())
        .filter((s) => s.length > 0);

    if (domains.length === 0 || ipsets.length === 0) {
        return null;
    }

    return { domains, ipsets };
};

/**
 * Format IPSetRule back to string
 */
export const formatIPSetRule = (rule: IPSetRule): string => {
    return `${rule.domains.join(',')}/${rule.ipsets.join(',')}`;
};

/**
 * Validate domain name or wildcard pattern
 */
export const validateDomain = (domain: string): string | undefined => {
    if (!domain || domain.trim().length === 0) {
        return 'Domain cannot be empty';
    }

    const trimmed = domain.trim();

    // Basic domain validation (allows wildcards like *.example.com)
    const domainRegex = /^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/;

    if (!domainRegex.test(trimmed)) {
        return 'Invalid domain format';
    }

    return undefined;
};

/**
 * Validate ipset name
 */
export const validateIPSetName = (name: string): string | undefined => {
    if (!name || name.trim().length === 0) {
        return 'IPSet name cannot be empty';
    }

    const trimmed = name.trim();

    // IPSet names: letters, digits, underscore, hyphen
    const ipsetNameRegex = /^[a-zA-Z0-9_-]+$/;

    if (!ipsetNameRegex.test(trimmed)) {
        return 'Invalid IPSet name (only letters, digits, _, - allowed)';
    }

    return undefined;
};

/**
 * Validate complete ipset rule string
 */
export const validateIPSetRule = (rule: string): string | undefined => {
    const trimmedRule = rule.trim();

    if (!trimmedRule) {
        return 'Rule cannot be empty';
    }

    const parsed = parseIPSetRule(trimmedRule);

    if (!parsed) {
        return 'Invalid rule format. Expected: DOMAIN[,DOMAIN,...]/IPSET_NAME[,IPSET_NAME,...]';
    }

    // Validate each domain
    const invalidDomain = parsed.domains.find((domain) => validateDomain(domain) !== undefined);
    if (invalidDomain) {
        const domainError = validateDomain(invalidDomain);
        return `Domain "${invalidDomain}": ${domainError}`;
    }

    // Validate each ipset name
    const invalidIpset = parsed.ipsets.find((ipset) => validateIPSetName(ipset) !== undefined);
    if (invalidIpset) {
        const ipsetError = validateIPSetName(invalidIpset);
        return `IPSet "${invalidIpset}": ${ipsetError}`;
    }

    return undefined;
};

/**
 * Validate domains input (comma-separated)
 */
export const validateDomainsInput = (input: string): string | undefined => {
    const trimmed = input.trim();

    if (!trimmed) {
        return 'At least one domain is required';
    }

    const domains = trimmed.split(',').map((d) => d.trim()).filter((d) => d.length > 0);

    if (domains.length === 0) {
        return 'At least one domain is required';
    }

    const invalidDomain = domains.find((domain) => validateDomain(domain) !== undefined);
    if (invalidDomain) {
        const error = validateDomain(invalidDomain);
        return `Domain "${invalidDomain}": ${error}`;
    }

    return undefined;
};

/**
 * Validate ipsets input (comma-separated)
 */
export const validateIPSetsInput = (input: string): string | undefined => {
    const trimmed = input.trim();

    if (!trimmed) {
        return 'At least one IPSet name is required';
    }

    const ipsets = trimmed.split(',').map((s) => s.trim()).filter((s) => s.length > 0);

    if (ipsets.length === 0) {
        return 'At least one IPSet name is required';
    }

    const invalidIpset = ipsets.find((ipset) => validateIPSetName(ipset) !== undefined);
    if (invalidIpset) {
        const error = validateIPSetName(invalidIpset);
        return `IPSet "${invalidIpset}": ${error}`;
    }

    return undefined;
};

/**
 * Check if rule is duplicate
 */
export const isDuplicateRule = (rule: string, existingRules: string[]): boolean => {
    const trimmedRule = rule.trim();
    return existingRules.some((r) => r.trim() === trimmedRule);
};
