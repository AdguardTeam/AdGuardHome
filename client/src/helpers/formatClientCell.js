import React, { Fragment } from 'react';
import { getClientInfo, getAutoClientInfo, normalizeWhois } from './helpers';
import { WHOIS_ICONS } from './constants';

const getFormattedWhois = (whois, t) => {
    const whoisInfo = normalizeWhois(whois);
    return (
        Object.keys(whoisInfo).map((key) => {
            const icon = WHOIS_ICONS[key];
            return (
                <span className="logs__whois text-muted" key={key} title={t(key)}>
                    {icon && (
                        <Fragment>
                            <svg className="logs__whois-icon icons">
                                <use xlinkHref={`#${icon}`} />
                            </svg>&nbsp;
                        </Fragment>
                    )}{whoisInfo[key]}
                </span>
            );
        })
    );
};

export const formatClientCell = (value, clients, autoClients, t) => {
    const clientInfo = getClientInfo(clients, value) || getAutoClientInfo(autoClients, value);
    const { name, whois } = clientInfo;
    let whoisContainer = '';
    let nameContainer = value;

    if (name) {
        nameContainer = (
            <span className="logs__text logs__text--wrap" title={`${name} (${value})`}>
                {name} <small>({value})</small>
            </span>
        );
    }

    if (whois) {
        whoisContainer = (
            <div className="logs__text logs__text--wrap logs__text--whois">
                {getFormattedWhois(whois, t)}
            </div>
        );
    }

    return (
        <span className="logs__text">
            <Fragment>
                {nameContainer}
                {whoisContainer}
            </Fragment>
        </span>
    );
};
