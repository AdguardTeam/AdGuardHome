import React from 'react';

import { Field, reduxForm } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import { renderInputField } from '../../../helpers/form';
import { validateAnswer, validateDomain, validateRequiredValue } from '../../../helpers/validators';
import { FORM_NAME } from '../../../helpers/constants';

interface FormProps {
    pristine: boolean;
    handleSubmit: (...args: unknown[]) => string;
    reset: (...args: unknown[]) => string;
    toggleRewritesModal: (...args: unknown[]) => unknown;
    submitting: boolean;
    processingAdd: boolean;
    t: (...args: unknown[]) => string;
    initialValues?: object;
}

const Form = (props: FormProps) => {
    const { t, handleSubmit, reset, pristine, submitting, toggleRewritesModal, processingAdd } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="modal-body">
                <div className="form__desc form__desc--top">
                    <Trans>domain_desc</Trans>
                </div>
                <div className="form__group">
                    <Field
                        id="domain"
                        name="domain"
                        component={renderInputField}
                        type="text"
                        className="form-control"
                        placeholder={t('form_domain')}
                        validate={[validateRequiredValue, validateDomain]}
                    />
                </div>
                <Trans>examples_title</Trans>:
                <ol className="leading-loose">
                    <li>
                        <code>example.org</code> – <Trans>example_rewrite_domain</Trans>
                    </li>

                    <li>
                        <code>*.example.org</code> –&nbsp;
                        <span>
                            <Trans components={[<code key="0">text</code>]}>example_rewrite_wildcard</Trans>
                        </span>
                    </li>
                </ol>
                <div className="form__group">
                    <Field
                        id="answer"
                        name="answer"
                        component={renderInputField}
                        type="text"
                        className="form-control"
                        placeholder={t('form_answer')}
                        validate={[validateRequiredValue, validateAnswer]}
                    />
                </div>
            </div>

            <ul>
                {['rewrite_ip_address', 'rewrite_domain_name', 'rewrite_A', 'rewrite_AAAA'].map((str) => (
                    <li key={str}>
                        <Trans components={[<code key="0">text</code>]}>{str}</Trans>
                    </li>
                ))}
            </ul>

            <div className="modal-footer">
                <div className="btn-list">
                    <button
                        type="button"
                        className="btn btn-secondary btn-standard"
                        disabled={submitting || processingAdd}
                        onClick={() => {
                            reset();
                            toggleRewritesModal();
                        }}>
                        <Trans>cancel_btn</Trans>
                    </button>

                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={submitting || pristine || processingAdd}>
                        <Trans>save_btn</Trans>
                    </button>
                </div>
            </div>
        </form>
    );
};

export default flow([
    withTranslation(),
    reduxForm({
        form: FORM_NAME.REWRITES,
        enableReinitialize: true,
    }),
])(Form);
