import React, { useRef } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, useTranslation } from 'react-i18next';
import classnames from 'classnames';
import Examples from './Examples';
import { renderRadioField, renderTextareaField, CheckboxField } from '../../../../helpers/form';
import {
    DNS_REQUEST_OPTIONS,
    FORM_NAME,
    UPSTREAM_CONFIGURATION_WIKI_LINK,
} from '../../../../helpers/constants';
import { testUpstreamWithFormValues } from '../../../../actions';
import { removeEmptyLines, trimLinesAndRemoveEmpty } from '../../../../helpers/helpers';
import { getTextareaCommentsHighlight, syncScroll } from '../../../../helpers/highlightTextareaComments';
import '../../../ui/texareaCommentsHighlight.css';

const UPSTREAM_DNS_NAME = 'upstream_dns';
const UPSTREAM_MODE_NAME = 'upstream_mode';

const renderField = ({
    name, component, type, className, placeholder,
    subtitle, value, normalizeOnBlur, containerClass, onScroll,
}) => {
    const { t } = useTranslation();
    const processingTestUpstream = useSelector((state) => state.settings.processingTestUpstream);
    const processingSetConfig = useSelector((state) => state.dnsConfig.processingSetConfig);

    return (
        <div
            key={placeholder}
            className={classnames('col-12 mb-4', containerClass)}
        >
            <Field
                id={name}
                value={value}
                name={name}
                component={component}
                type={type}
                className={className}
                placeholder={t(placeholder)}
                subtitle={t(subtitle)}
                disabled={processingSetConfig || processingTestUpstream}
                normalizeOnBlur={normalizeOnBlur}
                onScroll={onScroll}
            />
        </div>
    );
};

renderField.propTypes = {
    name: PropTypes.string.isRequired,
    component: PropTypes.element.isRequired,
    type: PropTypes.string.isRequired,
    className: PropTypes.string,
    placeholder: PropTypes.string.isRequired,
    subtitle: PropTypes.string,
    value: PropTypes.string,
    normalizeOnBlur: PropTypes.func,
    containerClass: PropTypes.string,
    onScroll: PropTypes.func,
};

const renderTextareaWithHighlightField = (props) => {
    const upstream_dns = useSelector((store) => store.form[FORM_NAME.UPSTREAM].values.upstream_dns);
    const upstream_dns_file = useSelector((state) => state.dnsConfig.upstream_dns_file);
    const ref = useRef(null);

    const onScroll = (e) => syncScroll(e, ref);

    return <>
        {renderTextareaField({
            ...props,
            disabled: !!upstream_dns_file,
            onScroll,
            normalizeOnBlur: trimLinesAndRemoveEmpty,
        })}
        {getTextareaCommentsHighlight(ref, upstream_dns)}
    </>;
};

renderTextareaWithHighlightField.propTypes = {
    className: PropTypes.string.isRequired,
    disabled: PropTypes.bool,
    id: PropTypes.string.isRequired,
    input: PropTypes.object,
    meta: PropTypes.object,
    normalizeOnBlur: PropTypes.func,
    onScroll: PropTypes.func,
    placeholder: PropTypes.string.isRequired,
    type: PropTypes.string.isRequired,
};

const INPUT_FIELDS = [
    {
        name: UPSTREAM_MODE_NAME,
        type: 'radio',
        value: DNS_REQUEST_OPTIONS.LOAD_BALANCING,
        component: renderRadioField,
        subtitle: 'load_balancing_desc',
        placeholder: 'load_balancing',
    },
    {
        name: UPSTREAM_MODE_NAME,
        type: 'radio',
        value: DNS_REQUEST_OPTIONS.PARALLEL,
        component: renderRadioField,
        subtitle: 'upstream_parallel',
        placeholder: 'parallel_requests',
    },
    {
        name: UPSTREAM_MODE_NAME,
        type: 'radio',
        value: DNS_REQUEST_OPTIONS.FASTEST_ADDR,
        component: renderRadioField,
        subtitle: 'fastest_addr_desc',
        placeholder: 'fastest_addr',
    },
];

const Form = ({
    submitting, invalid, handleSubmit,
}) => {
    const dispatch = useDispatch();
    const { t } = useTranslation();
    const upstream_dns = useSelector((store) => store.form[FORM_NAME.UPSTREAM].values.upstream_dns);
    const processingTestUpstream = useSelector((state) => state.settings.processingTestUpstream);
    const processingSetConfig = useSelector((state) => state.dnsConfig.processingSetConfig);
    const defaultLocalPtrUpstreams = useSelector(
        (state) => state.dnsConfig.default_local_ptr_upstreams,
    );

    const handleUpstreamTest = () => dispatch(testUpstreamWithFormValues());

    const testButtonClass = classnames('btn btn-primary btn-standard mr-2', {
        'btn-loading': processingTestUpstream,
    });

    const components = {
        a: <a href={UPSTREAM_CONFIGURATION_WIKI_LINK} target="_blank"
              rel="noopener noreferrer" />,
    };

    return <form onSubmit={handleSubmit} className="form--upstream">
        <div className="row">
            <label className="col form__label" htmlFor={UPSTREAM_DNS_NAME}>
                <Trans components={components}>upstream_dns_help</Trans>
                {' '}
                <Trans components={[
                    <a
                        href="https://link.adtidy.org/forward.html?action=dns_kb_providers&from=ui&app=home"
                        target="_blank"
                        rel="noopener noreferrer"
                        key="0"
                    >
                        DNS providers
                    </a>,
                ]}>
                    dns_providers
                </Trans>
            </label>
            <div className="col-12 mb-4">
                <div className="text-edit-container">
                    <Field
                        id={UPSTREAM_DNS_NAME}
                        name={UPSTREAM_DNS_NAME}
                        component={renderTextareaWithHighlightField}
                        type="text"
                        className="form-control form-control--textarea font-monospace text-input"
                        placeholder={t('upstream_dns')}
                        disabled={processingSetConfig || processingTestUpstream}
                        normalizeOnBlur={removeEmptyLines}
                    />
                </div>
            </div>
            {INPUT_FIELDS.map(renderField)}
            <div className="col-12">
                <Examples />
                <hr />
            </div>
            <div className="col-12">
                <label
                    className="form__label form__label--with-desc"
                    htmlFor="fallback_dns"
                >
                    <Trans>fallback_dns_title</Trans>
                </label>
                <div className="form__desc form__desc--top">
                    <Trans>fallback_dns_desc</Trans>
                </div>
                <Field
                    id="fallback_dns"
                    name="fallback_dns"
                    component={renderTextareaField}
                    type="text"
                    className="form-control form-control--textarea form-control--textarea-small font-monospace"
                    placeholder={t('fallback_dns_placeholder')}
                    disabled={processingSetConfig}
                    normalizeOnBlur={removeEmptyLines}
                />
            </div>
            <div className="col-12">
                <hr />
            </div>
            <div className="col-12 mb-2">
                <label
                    className="form__label form__label--with-desc"
                    htmlFor="bootstrap_dns"
                >
                    <Trans>bootstrap_dns</Trans>
                </label>
                <div className="form__desc form__desc--top">
                    <Trans>bootstrap_dns_desc</Trans>
                </div>
                <Field
                    id="bootstrap_dns"
                    name="bootstrap_dns"
                    component={renderTextareaField}
                    type="text"
                    className="form-control form-control--textarea form-control--textarea-small font-monospace"
                    placeholder={t('bootstrap_dns')}
                    disabled={processingSetConfig}
                    normalizeOnBlur={removeEmptyLines}
                />
            </div>
            <div className="col-12">
                <hr />
            </div>
            <div className="col-12">
                <label
                    className="form__label form__label--with-desc"
                    htmlFor="local_ptr"
                >
                    <Trans>local_ptr_title</Trans>
                </label>
                <div className="form__desc form__desc--top">
                    <Trans>local_ptr_desc</Trans>
                </div>
                <div className="form__desc form__desc--top">
                    {/** TODO: Add internazionalization for "" */}
                    {defaultLocalPtrUpstreams?.length > 0 ? (
                        <Trans values={{ ip: defaultLocalPtrUpstreams.map((s) => `"${s}"`).join(', ') }}>local_ptr_default_resolver</Trans>
                    ) : (
                        <Trans>local_ptr_no_default_resolver</Trans>
                    )}
                </div>
                <Field
                    id="local_ptr_upstreams"
                    name="local_ptr_upstreams"
                    component={renderTextareaField}
                    type="text"
                    className="form-control form-control--textarea form-control--textarea-small font-monospace"
                    placeholder={t('local_ptr_placeholder')}
                    disabled={processingSetConfig}
                    normalizeOnBlur={removeEmptyLines}
                />
                <div className="mt-4">
                    <Field
                        name="use_private_ptr_resolvers"
                        type="checkbox"
                        component={CheckboxField}
                        placeholder={t('use_private_ptr_resolvers_title')}
                        subtitle={t('use_private_ptr_resolvers_desc')}
                        disabled={processingSetConfig}
                    />
                </div>
            </div>
            <div className="col-12">
                <hr />
            </div>
            <div className="col-12 mb-4">
                <Field
                    name="resolve_clients"
                    type="checkbox"
                    component={CheckboxField}
                    placeholder={t('resolve_clients_title')}
                    subtitle={t('resolve_clients_desc')}
                    disabled={processingSetConfig}
                />
            </div>
        </div>
        <div className="card-actions">
            <div className="btn-list">
                <button
                    type="button"
                    className={testButtonClass}
                    onClick={handleUpstreamTest}
                    disabled={!upstream_dns || processingTestUpstream}
                >
                    <Trans>test_upstream_btn</Trans>
                </button>
                <button
                    type="submit"
                    className="btn btn-success btn-standard"
                    disabled={
                        submitting || invalid || processingSetConfig || processingTestUpstream
                    }
                >
                    <Trans>apply_btn</Trans>
                </button>
            </div>
        </div>
    </form>;
};

Form.propTypes = {
    handleSubmit: PropTypes.func,
    submitting: PropTypes.bool,
    invalid: PropTypes.bool,
    initialValues: PropTypes.object,
    upstream_dns: PropTypes.string,
    fallback_dns: PropTypes.string,
    bootstrap_dns: PropTypes.string,
};

export default reduxForm({ form: FORM_NAME.UPSTREAM })(Form);
