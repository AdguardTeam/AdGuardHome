export interface BlockedService {
    /** The SVG icon as a Base64-encoded string to make it easier to embed it into a data URL. */
    icon_svg: string;
    /** The ID of this service. */
    id: string;
    /** The human-readable name of this service. */
    name: string;
    /** The array of the filtering rules. */
    rules: string[];
    /** The ID of the group, that the service belongs to. */
    group_id?: string;
}
