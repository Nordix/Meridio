// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require('prism-react-renderer/themes/github');
const darkCodeTheme = require('prism-react-renderer/themes/dracula');

/** @type {import('@docusaurus/types').Config} */
const config = {
    title: 'Meridio',
    tagline: 'Facilitator of attraction and distribution of external traffic within Kubernetes via secondary networks',
    url: 'https://meridio.nordix.org/',
    baseUrl: '/',
    onBrokenLinks: 'throw',
    onBrokenMarkdownLinks: 'warn',
    favicon: 'img/favicon.ico',

    // GitHub pages deployment config.
    // If you aren't using GitHub pages, you don't need these.
    organizationName: 'nordix', // Usually your GitHub org/user name.
    projectName: 'meridio', // Usually your repo name.

    // Even if you don't use internalization, you can use this field to set useful
    // metadata like html lang. For example, if your site is Chinese, you may want
    // to replace "en" with "zh-Hans".
    i18n: {
        defaultLocale: 'en',
        locales: ['en'],
    },

    staticDirectories: ['../docs/resources'],

    presets: [
        [
            'classic',
            /** @type {import('@docusaurus/preset-classic').Options} */
            ({
                docs: {
                    sidebarPath: require.resolve('./sidebars.js'),
                    path: "../docs",
                    // Please change this to your repo.
                    // Remove this to remove the "edit this page" links.
                    editUrl: 'https://github.com/Nordix/Meridio/tree/master/docs/',
                    lastVersion: 'current',
                    versions: {
                        current: {
                            label: 'latest',
                            // path: '',
                            banner: 'none',
                        },
                        "v1.0.0": {
                            label: 'v1.0.0',
                            path: '/v1.0.0',
                            banner: 'none',
                        },
                    },
                },
                theme: {
                    customCss: require.resolve('./src/css/custom.css'),
                },
                sitemap: {
                    changefreq: 'weekly',
                    priority: 0.5,
                    ignorePatterns: ['/tags/**'],
                    filename: 'sitemap.xml',
                },
            }),
        ],
    ],

    themeConfig:
        /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
        ({
            navbar: {
                title: 'Meridio',
                logo: {
                    alt: 'Meridio Logo',
                    src: 'Logo.svg',
                },
                items: [
                    {
                        type: 'docsVersionDropdown',
                        position: 'left',
                        dropdownActiveClassDisabled: true,
                    },
                    {
                        type: 'doc',
                        docId: 'overview',
                        position: 'left',
                        label: 'Documentation',
                    },
                    {
                        href: 'https://github.com/nordix/meridio',
                        position: 'right',
                        className: 'header-github-link header-icon-link',
                    },
                ],
            },
            algolia: {
                appId: 'E15FFWY7MY',
                apiKey: '801c089814478d0030a1f4f60615b715',
                indexName: 'meridio-nordix',
            },
            announcementBar: {
                id: 'meridio-github-star',
                content: `⭐️ If you like Meridio, give it a star on <a target="_blank" rel="noopener noreferrer" href="https://github.com/Nordix/Meridio">GitHub</a>`,
                backgroundColor: '#F2F7FF',
            },
            footer: {
                style: 'dark',
                links: [
                    {
                        title: 'Docs',
                        items: [
                            {
                                label: 'Overview',
                                to: '/docs/overview',
                            },
                            {
                                label: 'Frequently Asked Questions',
                                to: '/docs/faq',
                            },
                        ],
                    },
                    {
                        title: 'Community',
                        items: [
                            {
                                label: 'Youtube',
                                href: 'https://www.youtube.com/channel/UCh8IioW7F3nXdBOZyyLIxRQ',
                            },
                            {
                                label: 'Slack',
                                href: 'https://cloud-native.slack.com/archives/C03ETG3J04S',
                            },
                        ],
                    },
                    {
                        title: 'More',
                        items: [
                            {
                                label: 'GitHub',
                                href: 'https://github.com/nordix/meridio',
                            },
                        ],
                    },
                ],
                copyright: `Copyright © ${new Date().getFullYear()} Nordix, Inc. Built with Docusaurus.`,
            },
            prism: {
                theme: lightCodeTheme,
                darkTheme: darkCodeTheme,
            },
        }),
};

module.exports = config;
