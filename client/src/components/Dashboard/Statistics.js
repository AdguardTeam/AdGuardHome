import React from 'react';
import PropTypes from 'prop-types';
import { withNamespaces, Trans } from 'react-i18next';

import StatsCard from './StatsCard';
import { getPercent, normalizeHistory } from '../../helpers/helpers';

const getNormalizedHistory = (data, interval, id) => [
    { data: normalizeHistory(data, interval), id },
];

const Statistics = ({
    interval,
    dnsQueries,
    blockedFiltering,
    replacedSafebrowsing,
    replacedParental,
    numDnsQueries,
    numBlockedFiltering,
    numReplacedSafebrowsing,
    numReplacedParental,
}) => (
    <div className="row">
        <div className="col-sm-6 col-lg-3">
            <StatsCard
                total={numDnsQueries}
                lineData={getNormalizedHistory(dnsQueries, interval, 'dnsQuery')}
                title={<Trans>dns_query</Trans>}
                color="blue"
            />
        </div>
        <div className="col-sm-6 col-lg-3">
            <StatsCard
                total={numBlockedFiltering}
                lineData={getNormalizedHistory(blockedFiltering, interval, 'blockedFiltering')}
                percent={getPercent(numDnsQueries, numBlockedFiltering)}
                title={<Trans components={[<a href="#filters" key="0">link</a>]}>blocked_by</Trans>}
                color="red"
            />
        </div>
        <div className="col-sm-6 col-lg-3">
            <StatsCard
                total={numReplacedSafebrowsing}
                lineData={getNormalizedHistory(
                    replacedSafebrowsing,
                    interval,
                    'replacedSafebrowsing',
                )}
                percent={getPercent(numDnsQueries, numReplacedSafebrowsing)}
                title={<Trans>stats_malware_phishing</Trans>}
                color="green"
            />
        </div>
        <div className="col-sm-6 col-lg-3">
            <StatsCard
                total={numReplacedParental}
                lineData={getNormalizedHistory(replacedParental, interval, 'replacedParental')}
                percent={getPercent(numDnsQueries, numReplacedParental)}
                title={<Trans>stats_adult</Trans>}
                color="yellow"
            />
        </div>
    </div>
);

Statistics.propTypes = {
    interval: PropTypes.number.isRequired,
    dnsQueries: PropTypes.array.isRequired,
    blockedFiltering: PropTypes.array.isRequired,
    replacedSafebrowsing: PropTypes.array.isRequired,
    replacedParental: PropTypes.array.isRequired,
    numDnsQueries: PropTypes.number.isRequired,
    numBlockedFiltering: PropTypes.number.isRequired,
    numReplacedSafebrowsing: PropTypes.number.isRequired,
    numReplacedParental: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
};

export default withNamespaces()(Statistics);
