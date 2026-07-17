/**
 * Protection state configuration
 */
export interface SetProtectionRequest {
    enabled: boolean;
    /** Duration of a pause, in milliseconds.  Enabled should be false. */
    duration?: number;
}
