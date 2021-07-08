module.exports = {
    title: 'goInception',
    base:"/goInception/",
    locales: {
      // 键名是该语言所属的子路径
      // 作为特例，默认语言可以使用 '/' 作为其路径。
      '/': {
        lang: 'en-US', // 将会被设置为 <html> 的 lang 属性
        title: 'goInception Docs',
        description: 'goInception extension of the usage of Inception, to specify the remote server by adding annotations before the SQL review, and for distinguishing SQL and review adding special comments at the beginning and the end of SQL.'
      },
      '/zh/': {
        lang: 'zh-CN',
        title: 'goInception使用文档',
        description: 'goInception是一个集审核、执行、备份及生成回滚语句于一身的MySQL运维工具'
      }
    },
    themeConfig: {
        repo: 'hanchuanchuan/goInception',
        editLinks: true,
        docsDir: 'docs',
        lastUpdated: 'Last Updated',

        locales: {
          '/': {
            selectText: 'Languages',
            label: 'English',
            ariaLabel: 'Languages',
            editLinkText: 'Edit this page on GitHub',
            serviceWorker: {
              updatePopup: {
                message: "New content is available.",
                buttonText: "Refresh"
              }
            },
            algolia: {},
            // nav:
            // [
            //     { text: 'External', link: 'https://github.com/hanchuanchuan/goInception/stargazers',src:"https://img.shields.io/github/stars/hanchuanchuan/goInception", target:'_self', rel:'' },

            //     <a href="https://github.com/hanchuanchuan/goInception/stargazers"><img alt="GitHub stars" src="https://img.shields.io/github/stars/hanchuanchuan/goInception"></a>

            //     [![GitHub stars](https://img.shields.io/github/stars/hanchuanchuan/goInception)](https://github.com/hanchuanchuan/goInception/stargazers)
            // ],
            sidebar: {
                "/": [
                    ["","Introdunction"],
                    {
                        title: 'Start',
                        collapsable: false,
                        sidebarDepth: 1,
                        children: [
                            ["install.md","Install"],
                            ["params.md","Params"],
                            ["demo.md","Demo"],
                            ["result.md","Result Desc"],
                            ["config.md","config.toml Desc"],
                            ["permission.md","Permission Desc"],
                        ]
                    },
                    {
                        title: 'Config',
                        collapsable: false,
                        sidebarDepth: 1,
                        children: [
                            ["options.md","Options"],
                            ["rules.md","Rules"],
                            ["backup.md","Backup"],
                            ["osc.md","Online-DDL: pt-osc"],
                            ["ghost.md","Online-DDL: gh-ost"],
                        ]
                    },
                    {
                        title: 'Advanced',
                        collapsable: false,
                        sidebarDepth: 1,
                        children: [
                            ["levels.md","Custom audit level"],
                            ["kill_stmt.md","Kill Operation"],
                            ["statistics.md","Statistics"],
                            ["tree.md","Syntax tree"],
                            ["trans.md","Transaction"],
                            ["safe.md","User authentication"],
                            ["diff.md","VS Inception"],
                        ]
                    },
                    ["support.md","Support"],
                    ["changelog.md","Changelog"],
                ],
            },
          },
          '/zh/': {
            selectText: '选择语言',
            label: '简体中文',
            editLinkText: '在 GitHub 上编辑此页',
            // Service Worker 的配置
            serviceWorker: {
              updatePopup: {
                message: "发现新内容可用.",
                buttonText: "刷新"
              }
            },
            // 当前 locale 的 algolia docsearch 选项
            algolia: {},
            sidebar: {
                "/zh/": [
                    ["","介绍"],
                    {
                        title: '开始',
                        collapsable: false,
                        sidebarDepth: 1,
                        children: [
                            ["install.md","安装"],
                            ["params.md","调用选项"],
                            ["demo.md","使用示例"],
                            ["result.md","结果集说明"],
                            ["config.md","config.toml说明"],
                            ["permission.md","权限说明"],
                        ]
                    },
                    {
                        title: '配置',
                        collapsable: false,
                        sidebarDepth: 1,
                        children: [
                            ["options.md","审核选项"],
                            ["rules.md","审核规则"],
                            ["backup.md","备份功能"],
                            ["osc.md","DDL变更:pt-osc"],
                            ["ghost.md","DDL变更:gh-ost"],
                        ]
                    },
                    {
                        title: '进阶',
                        collapsable: false,
                        sidebarDepth: 1,
                        children: [
                            ["levels.md","自定义审核级别"],
                            ["kill_stmt.md","KILL操作"],
                            ["statistics.md","统计功能"],
                            ["tree.md","语法树打印"],
                            ["trans.md","事务"],
                            ["safe.md","用户管理/鉴权"],
                            ["diff.md","对比inception"],
                        ]
                    },
                    ["support.md","赞助&定制"],
                    ["changelog.md","更新日志"],
                ],
            },
          }
        }
      },
  }
