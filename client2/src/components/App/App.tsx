import React, { FC } from 'react';
import { BrowserRouter } from 'react-router-dom';

import Icons from 'Common/ui/Icons';
import Routes from './Routes';

import { ErrorBoundary } from './Errors';

const App: FC = () => {
    return (
        <ErrorBoundary>
            <BrowserRouter>
                <Routes />
                <Icons />
            </BrowserRouter>
        </ErrorBoundary>
    );
};

export default App;
