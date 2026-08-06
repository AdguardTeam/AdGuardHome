/**
 * Information about the latest available version of AdGuard Home.
 */
export interface VersionInfo {
    /** If true then other fields doesn't appear. */
    disabled: boolean;
    new_version?: string;
    announcement?: string;
    announcement_url?: string;
    can_autoupdate?: boolean;
}
