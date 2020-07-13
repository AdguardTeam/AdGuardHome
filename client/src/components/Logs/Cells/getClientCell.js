import React from 'react';
import { nanoid } from 'nanoid';
import classNames from 'classnames';
import PropTypes from 'prop-types';
import { formatClientCell } from '../../../helpers/formatClientCell';
import getHintElement from './getHintElement';
import { checkFiltered } from '../../../helpers/helpers';
import { BLOCK_ACTIONS } from '../../../helpers/constants';

const getClientCell = ({
    row, t, isDetailed, toggleBlocking, autoClients, processingRules,
}) => {
    const {
        reason, client, domain, info: { name, whois_info },
    } = row.original;

    const autoClient = autoClients.find((autoClient) => autoClient.name === client);
    const source = autoClient?.source;

    const id = nanoid();

    const data = {
        address: client,
        name,
        country: whois_info?.country,
        city: whois_info?.city,
        network: whois_info?.orgname,
        source_label: source,
    };

    const processedData = Object.entries(data);

    const isFiltered = checkFiltered(reason);

    const nameClass = classNames('w-90 o-hidden d-flex flex-column', {
        'mt-2': isDetailed && !name,
        'white-space--nowrap': isDetailed,
    });

    const hintClass = classNames('icons mr-4 icon--small cursor--pointer icon--light-gray', {
        'my-3': isDetailed,
    });

    const renderBlockingButton = (isFiltered, domain) => {
        const buttonType = isFiltered ? BLOCK_ACTIONS.UNBLOCK : BLOCK_ACTIONS.BLOCK;

        const buttonClass = classNames('logs__action button__action', {
            'btn-outline-secondary': isFiltered,
            'btn-outline-danger': !isFiltered,
            'logs__action--detailed': isDetailed,
        });

        const onClick = () => toggleBlocking(buttonType, domain);

        return (
            <div className={buttonClass}>
                <button
                    type="button"
                    className={`btn btn-sm ${buttonClass}`}
                    onClick={onClick}
                    disabled={processingRules}
                >
                    {t(buttonType)}
                </button>
            </div>
        );
    };

    return (
        <div className="logs__row o-hidden h-100">
            {getHintElement({
                className: hintClass,
                columnClass: 'grid grid--limited',
                tooltipClass: 'px-5 pb-5 pt-4 mw-75',
                xlinkHref: 'question',
                contentItemClass: 'text-truncate key-colon',
                title: 'client_details',
                content: processedData,
                placement: 'bottom',
            })}
            <div
                className={nameClass}>
                <div data-tip={true} data-for={id}>{formatClientCell(row, t, isDetailed)}</div>
                {isDetailed && name
                && <div className="detailed-info d-none d-sm-block logs__text"
                        title={name}>{name}</div>}
            </div>
            {renderBlockingButton(isFiltered, domain)}
        </div>
    );
};

getClientCell.propTypes = {
    row: PropTypes.object.isRequired,
    t: PropTypes.func.isRequired,
    isDetailed: PropTypes.bool.isRequired,
    toggleBlocking: PropTypes.func.isRequired,
    autoClients: PropTypes.array.isRequired,
    processingRules: PropTypes.bool.isRequired,
};

export default getClientCell;
