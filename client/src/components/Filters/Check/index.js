import React from 'react';
import PropTypes from 'prop-types';
import { useTranslation } from 'react-i18next';
import { Field, reduxForm } from 'redux-form';
import { useSelector } from 'react-redux';
import Card from '../../ui/Card';
import { renderInputField } from '../../../helpers/form';
import Info from './Info';
import { FORM_NAME } from '../../../helpers/constants';

const Check = (props) => {
    const {
        pristine,
        invalid,
        handleSubmit,
    } = props;

    const { t } = useTranslation();

    const processingCheck = useSelector((state) => state.filtering.processingCheck);
    const hostname = useSelector((state) => state.filtering.check.hostname);

    return <Card
        title={t('check_title')}
        subtitle={t('check_desc')}
    >
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-12 col-md-6">
                    <div className="input-group">
                        <Field
                            id="name"
                            name="name"
                            component={renderInputField}
                            type="text"
                            className="form-control"
                            placeholder={t('form_enter_host')}
                        />
                        <span className="input-group-append">
                            <button
                                className="btn btn-success btn-standard btn-large"
                                type="submit"
                                onClick={handleSubmit}
                                disabled={pristine || invalid || processingCheck}
                            >
                                {t('check')}
                            </button>
                        </span>
                    </div>
                    {hostname && <>
                        <hr />
                        <Info />
                    </>}
                </div>
            </div>
        </form>
    </Card>;
};

Check.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    pristine: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
};

export default reduxForm({ form: FORM_NAME.DOMAIN_CHECK })(Check);
