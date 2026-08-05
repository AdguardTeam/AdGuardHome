// Converts an option value to string; null/undefined → '' (not "null"/"undefined").
export const optionToValue = (value: unknown): string => (value == null ? '' : String(value));

// Case-insensitive label filter; empty query returns options unchanged.
export const filterOptions = <T extends { label: unknown }>(options: T[], query: string): T[] => {
    const q = query.toLowerCase();
    if (!q) return options;
    return options.filter((opt) => String(opt.label).toLowerCase().includes(q));
};

// Returns `${prefix}-${value}` for test IDs, or undefined when no prefix.
export const getItemTestId = (prefix: string | undefined, value: unknown): string | undefined =>
    prefix ? `${prefix}-${optionToValue(value)}` : undefined;
