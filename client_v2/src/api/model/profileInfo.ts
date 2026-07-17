import type { Lang } from './lang';
import type { ProfileInfoTheme } from './profileInfoTheme';

/**
 * Information about the current user
 */
export interface ProfileInfo {
    name: string;
    language: Lang;
    /** Interface theme */
    theme: ProfileInfoTheme;
}
