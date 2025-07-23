/**
 * Helper type utilities for Storybook stories
 */

/**
 * Type assertion helper to suppress TypeScript errors in story args
 * Use this when the component props and Storybook's typing don't align perfectly
 */
export const asStoryArgs = <T>(args: any): T => args as T;
