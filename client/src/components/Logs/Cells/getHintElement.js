import React from 'react';
import PropTypes from 'prop-types';
import TooltipTrigger from 'react-popper-tooltip';
import { Trans } from 'react-i18next';
import classNames from 'classnames';
import './Tooltip.css';
import 'react-popper-tooltip/dist/styles.css';
import { HIDE_TOOLTIP_DELAY } from '../../../helpers/constants';
import { processContent } from '../../../helpers/helpers';

const getHintElement = ({
    className,
    contentItemClass,
    columnClass,
    canShowTooltip = true,
    xlinkHref,
    title,
    placement,
    tooltipClass,
    content,
    renderContent = content ? React.Children.map(
        processContent(content),
        (item, idx) => <div key={idx} className={contentItemClass}>
            <Trans>{item || '—'}</Trans>
        </div>,
    ) : null,
}) => <TooltipTrigger placement={placement} trigger="hover" delayHide={HIDE_TOOLTIP_DELAY} tooltip={
    ({
        tooltipRef,
        getTooltipProps,
    }) => <div {...getTooltipProps({
        ref: tooltipRef,
        className: classNames('tooltip__container', tooltipClass, { 'd-none': !canShowTooltip }),
    })}
    >
        {title && <div className="pb-4 h-25 grid-content font-weight-bold">
            <Trans>{title}</Trans>
        </div>}
        <div className={classNames(columnClass)}>{renderContent}</div>
    </div>
}>{({
    getTriggerProps, triggerRef,
}) => <span {...getTriggerProps({ ref: triggerRef })}>
            {xlinkHref && <svg className={className}>
                <use xlinkHref={`#${xlinkHref}`} />
            </svg>}
  </span>}
</TooltipTrigger>;

getHintElement.propTypes = {
    className: PropTypes.string,
    contentItemClass: PropTypes.string,
    columnClass: PropTypes.string,
    tooltipClass: PropTypes.string,
    title: PropTypes.string,
    placement: PropTypes.string,
    canShowTooltip: PropTypes.string,
    xlinkHref: PropTypes.string,
    content: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.array,
    ]),
    renderContent: PropTypes.arrayOf(PropTypes.element),
};

export default getHintElement;
