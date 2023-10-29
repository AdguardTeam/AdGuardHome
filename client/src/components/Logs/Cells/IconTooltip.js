import React from 'react';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';
import classNames from 'classnames';
import { processContent } from '../../../helpers/helpers';
import Tooltip from '../../ui/Tooltip';
import 'react-popper-tooltip/dist/styles.css';
import './IconTooltip.css';
import { SHOW_TOOLTIP_DELAY } from '../../../helpers/constants';

const IconTooltip = ({
    className,
    contentItemClass,
    columnClass,
    triggerClass,
    canShowTooltip = true,
    xlinkHref,
    title,
    placement,
    tooltipClass,
    content,
    trigger,
    onVisibilityChange,
    defaultTooltipShown,
    delayHide,
    renderContent = content ? React.Children.map(
        processContent(content),
        (item, idx) => <div key={idx} className={contentItemClass}>
            <Trans>{item || '—'}</Trans>
        </div>,
    ) : null,
}) => {
    const tooltipContent = <>
        {title
        && <div className="pb-4 h-25 grid-content font-weight-bold"><Trans>{title}</Trans></div>}
        <div className={classNames(columnClass)}>{renderContent}</div>
    </>;

    const tooltipClassName = classNames('tooltip-custom__container', tooltipClass, { 'd-none': !canShowTooltip });

    return <Tooltip
        className={tooltipClassName}
        content={tooltipContent}
        placement={placement}
        triggerClass={triggerClass}
        trigger={trigger}
        onVisibilityChange={onVisibilityChange}
        delayShow={trigger === 'click' ? 0 : SHOW_TOOLTIP_DELAY}
        delayHide={delayHide}
        defaultTooltipShown={defaultTooltipShown}
    >
        {xlinkHref && <svg className={className}>
            <use xlinkHref={`#${xlinkHref}`} />
        </svg>}
    </Tooltip>;
};

IconTooltip.propTypes = {
    className: PropTypes.string,
    trigger: PropTypes.string,
    triggerClass: PropTypes.string,
    contentItemClass: PropTypes.string,
    columnClass: PropTypes.string,
    tooltipClass: PropTypes.string,
    title: PropTypes.string,
    placement: PropTypes.string,
    canShowTooltip: PropTypes.bool,
    xlinkHref: PropTypes.string,
    content: PropTypes.node,
    renderContent: PropTypes.arrayOf(PropTypes.element),
    onVisibilityChange: PropTypes.func,
    defaultTooltipShown: PropTypes.bool,
    delayHide: PropTypes.number,
};

export default IconTooltip;
