import React, { Fragment } from 'react';
import { getClientInfo } from './helpers';

export const formatClientCell = (value, clients, autoClients) => {
    const clientInfo = getClientInfo(clients, value) || getClientInfo(autoClients, value);
    const { name, whois } = clientInfo;

    if (whois && name) {
        return (
            <Fragment>
                <div className="logs__text logs__text--wrap" title={`${name} (${value})`}>
                    {name} <small className="text-muted-dark">({value})</small>
                </div>
                <div className="logs__text logs__text--wrap" title={whois}>
                    <small className="text-muted">{whois}</small>
                </div>
            </Fragment>
        );
    } else if (name) {
        return (
            <span className="logs__text logs__text--wrap" title={`${name} (${value})`}>
                {name} <small>({value})</small>
            </span>
        );
    }

    return (
        <span className="logs__text" title={value}>
            {value}
        </span>
    );
};
