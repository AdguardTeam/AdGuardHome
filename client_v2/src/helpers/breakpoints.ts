/**
 * Layout breakpoints used by the responsive hooks in
 * `panel/hooks/useMediaQuery`.
 *
 * CSS custom properties cannot be referenced inside `@media` queries, so the
 * same numbers are repeated by hand in the `.pcss` modules.  Keep the two in
 * sync when a breakpoint changes.
 */

/**
 * Minimum viewport width of the desktop layout.  At or above this width the
 * dashboard drops its mobile affordances and the data tables grow their
 * desktop-only columns.
 */
export const DESKTOP_MIN_WIDTH = 768;

/**
 * Exclusive upper bound of the mobile layout.  Below this width data tables
 * collapse into card lists.
 *
 * Intentionally wider than {@link DESKTOP_MIN_WIDTH}: the two hooks measure
 * different things, so viewports in between report as both "desktop" and
 * "mobile".
 */
export const MOBILE_MAX_WIDTH = 1024;
