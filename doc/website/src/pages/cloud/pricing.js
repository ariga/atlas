import React, {useEffect} from 'react';
import Layout from '@theme/Layout';
import {IntercomProvider, useIntercom } from 'react-use-intercom';


const LetsChat = () => {
    const { boot, shutdown, show } = useIntercom();

    useEffect(() => {
        boot()
        return () => {
            shutdown()
        }
    }, [])
    return (
        <div className="btn btn-primary btn-block p-2 shadow rounded-pill" onClick={() => show()}>
            Let's Chat
        </div>
    )
}

export default function Pricing() {
    return (
        <IntercomProvider appId={"ugatw0l1"}>
            <Layout>
                <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css"/>
                <link rel="stylesheet"
                      href="https://stackpath.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css"/>
                <section className={"pricing-section"}>
                    <div className="container py-5">
                        <header className="text-center mb-5 text-white">
                            <div className="row">
                                <div className="col-lg-7 mx-auto">
                                    <p>
                                        <img src="https://atlasgo.io/uploads/pricing/atlas-cloud-by-ariga.png" alt=""/>
                                    </p>
                                    <h1>Choose your plan</h1>
                                </div>
                            </div>
                        </header>

                        <div className="row text-center align-items-end">
                            <div className="col-lg-6 mb-6 mb-lg-0">
                                <div className="bg-white p-5 rounded-lg shadow">
                                    <h1 className="h6 text-uppercase font-weight-bold mb-4">Community</h1>

                                    <div className="custom-separator my-4 mx-auto bg-primary"></div>

                                    <ul className="list-unstyled my-5 text-small text-left">
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> 5 Migration Directories
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> 5 Seats
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> 500 Runs / month
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> Backward-compatibility
                                            protection
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> Destructive change
                                            detection
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> Schema change simulation
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> 30+ other schema checks
                                        </li>
                                        <li>
                                            <i className="fa fa-check mr-2 text-primary"></i> Community support
                                        </li>

                                    </ul>
                                    <a href="https://auth.atlasgo.cloud/signup"
                                       className="btn btn-primary btn-block p-2 shadow rounded-pill">
                                        Start for free
                                    </a>
                                </div>
                            </div>
                            <div className="col-lg-6 mb-6 mb-lg-0">
                                <div className="bg-white p-5 rounded-lg shadow">
                                    <h1 className="h6 text-uppercase font-weight-bold mb-4">Commercial</h1>

                                    <div className="custom-separator my-4 mx-auto bg-primary"></div>
                                    <p>
                                        Everything in the Community plan, plus:
                                    </p>
                                    <ul className="list-unstyled my-5 text-small text-left font-weight-normal">
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> Single sign-on (SSO)
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> Pre-deployment simulation
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> Schema change
                                            notifications
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> Unlimited runs
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> Deployments
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> Drift detection and
                                            centralized policies
                                        </li>
                                        <li className="mb-3">
                                            <i className="fa fa-check mr-2 text-primary"></i> Dedicated support
                                        </li>
                                    </ul>
                                    <LetsChat/>
                                </div>
                            </div>

                        </div>
                    </div>
                </section>
            </Layout>
        </IntercomProvider>
    );
}