import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { withTranslation, Trans } from 'react-i18next';

import StatsCard from './StatsCard';
import { getPercent, normalizeHistory } from '../../helpers/helpers';
import { RESPONSE_FILTER } from '../../helpers/constants';

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
                title={<Link to="logs"><Trans>dns_query</Trans></Link>}
                color="blue"
            />
        </div>
        <div className="col-sm-6 col-lg-3">
            <StatsCard
                total={numBlockedFiltering}
                lineData={getNormalizedHistory(blockedFiltering, interval, 'blockedFiltering')}
                percent={getPercent(numDnsQueries, numBlockedFiltering)}
                title={<Trans components={[<Link to={`logs?response_status=${RESPONSE_FILTER.BLOCKED.QUERY}`} key="0">link</Link>]}>blocked_by</Trans>}
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
                title={<Link to={`logs?response_status=${RESPONSE_FILTER.BLOCKED_THREATS.QUERY}`}><Trans>stats_malware_phishing</Trans></Link>}
                color="green"
            />
        </div>
        <div className="col-sm-6 col-lg-3">
            <StatsCard
                total={numReplacedParental}
                lineData={getNormalizedHistory(replacedParental, interval, 'replacedParental')}
                percent={getPercent(numDnsQueries, numReplacedParental)}
                title={<Link to={`logs?response_status=${RESPONSE_FILTER.BLOCKED_ADULT_WEBSITES.QUERY}`}><Trans>stats_adult</Trans></Link>}
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

export default withTranslation()(Statistics);
