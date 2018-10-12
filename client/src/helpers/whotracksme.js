import trackersDb from './whotracksmedb.json';

/**
  @typedef TrackerData
  @type {object}
  @property {string} id - tracker ID.
  @property {string} name - tracker name.
  @property {number} age - tracker category.
 */

/**
 * Gets tracker data in the whotracksme database
 *
 * @param {String} domainName domain name to check
 * @returns {TrackerData} tracker data or null if no matching tracker found
 */
export const getTrackerData = (domainName) => {
    if (!domainName) {
        return null;
    }

    const parts = domainName.split(/\./g).reverse();
    let hostToCheck = '';

    // Check every subdomain except the TLD
    for (let i = 1; i < parts.length; i += 1) {
        hostToCheck = hostToCheck + (i > 0 ? '.' : '') + parts[i];
        const trackerId = trackersDb.trackerDomains[hostToCheck];

        if (trackerId) {
            const trackerData = trackersDb.trackers[trackerId];
            const categoryName = trackersDb.categories[trackerData.categoryId];

            return {
                id: trackerId,
                name: trackerData.name,
                category: categoryName,
            };
        }
    }

    // No tracker found for the specified domain
    return null;
};
