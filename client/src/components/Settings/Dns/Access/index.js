import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withTranslation } from 'react-i18next';

import Form from './Form';
import Card from '../../../ui/Card';

class Access extends Component {
    handleFormSubmit = (values) => {
        this.props.setAccessList(values);
    };

    render() {
        const { t, access } = this.props;

        const { processing, processingSet, ...values } = access;

        return (
            <Card
                title={t('access_title')}
                subtitle={t('access_desc')}
                bodyType="card-body box-body--settings"
            >
                <Form
                    initialValues={values}
                    onSubmit={this.handleFormSubmit}
                    processingSet={processingSet}
                />
            </Card>
        );
    }
}

Access.propTypes = {
    access: PropTypes.object.isRequired,
    setAccessList: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withTranslation()(Access);
