import React from 'react';
import clsx from 'clsx';
import styles from './styles.module.css';

const FeatureList = [
    {
        title: 'Secondary Networking',
        Svg: require('@site/static/img/undraw_docusaurus_mountain.svg').default,
        description: (
            <>
                Isolation of the traffic and the network is a key aspect for Meridio, it improves the resiliency, the security, the decongestion and the minimization of the traffic. In addition, each network can have its own specificities and capabilities provided via different dataplane methods (VPP, OVS, Kernel, accelerated network...) which can carry exotic protocols. Meridio is for now, providing dataplane only for TCP, UDP and SCTP traffic on IPv4 and IPv6.
            </>
        ),
    },
    {
        title: 'External Traffic Attraction',
        Svg: require('@site/static/img/undraw_docusaurus_tree.svg').default,
        description: (
            <>
                Announcing service-addresses (VIP-addresses) to the external gateways permits the traffic to be attracted towards the different services exposed by the target applications. Frontend services provide different connectivity mechanisms such as VLANs or host network interfaces to connect the network services to the gateways. To announce the service and ensure the link-supervision, routing protocols are used. For now, BGP and Static BFD are supported.
            </>
        ),
    },
    {
        title: 'Network Services',
        Svg: require('@site/static/img/undraw_docusaurus_react.svg').default,
        description: (
            <>
                Development of new network services with more or different capabilities (network acceleration, SCTP load-balancing...) is possible within Meridio. As the current default network service, a no-NAT stateless Load-Balancer is offered by Meridio. It provides traffic classification (based on 5-tuple) to steer traffic into multiple different instances of the network service applications can subscribe to.
            </>
        ),
    },
    {
        title: 'Runtime Configuration',
        Svg: require('@site/static/img/undraw_docusaurus_react.svg').default,
        description: (
            <>
                Meridio users have the flexibility to adjust the network services on the fly as they desire. Traffic attractors, streams gathering traffic into logical groups and traffic classifiers (flows) can be added, removed and updated without any redeployment of the resources, and with no traffic disturbance. Individually, each user pods have the ability to select the traffic to consume at runtime which will produce secondary network reorganization to cover the user pods needs and requests.
            </>
        ),
    },
];

function Feature({ Svg, title, description }) {
    return (
        <div className={clsx('col col--3')}>
            <div className="text--center">
                <Svg className={styles.featureSvg} role="img" />
            </div>
            <div className="text--center padding-horiz--md">
                <h3>{title}</h3>
                <p>{description}</p>
            </div>
        </div>
    );
}

export default function HomepageFeatures() {
    return (
        <section className={styles.features}>
            <div className="container">
                <div className="row">
                    {FeatureList.map((props, idx) => (
                        <Feature key={idx} {...props} />
                    ))}
                </div>
            </div>
        </section>
    );
}
