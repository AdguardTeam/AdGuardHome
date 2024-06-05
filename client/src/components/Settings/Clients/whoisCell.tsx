import React, { Fragment } from 'react';

import { normalizeWhois } from '../../../helpers/helpers';
import { WHOIS_ICONS } from '../../../helpers/constants';

const getFormattedWhois = (value: any, t: any) => {
    const whoisInfo = normalizeWhois(value);
    const whoisKeys = Object.keys(whoisInfo);

    if (whoisKeys.length > 0) {
        return whoisKeys.map((key) => {
            const icon = WHOIS_ICONS[key];
            return (
                <div key={key} title={t(key)}>
                    {icon && (
                        <Fragment>
                            <svg className="logs__whois-icon text-muted-dark icons icon--24">
                                <use xlinkHref={`#${icon}`} />
                            </svg>
                            &nbsp;
                        </Fragment>
                    )}
                    {whoisInfo[key]}
                </div>
            );
        });
    }

    return '–';
};

const whoisCell = (t: any) =>
    function cell(row: any) {
        const { value } = row;

        return (
            <div className="logs__row o-hidden">
                <div className="logs__text logs__text--wrap">{getFormattedWhois(value, t)}</div>
            </div>
        );
    };

export default whoisCell;
