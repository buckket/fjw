<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:content="http://purl.org/rss/1.0/modules/content/">
    <xsl:preserve-space elements="*" />
    <xsl:output method="html" encoding="UTF-8" indent="yes" doctype-system="about:legacy-compat"/>
    <xsl:template match="/">
        <html lang="de">
            <head>
                <title>Post von Wagner</title>
                <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/water.css@2/out/light.css"/>
                <link rel="alternate" type="application/rss+xml" title="RSS"
                      href="https://buckket.github.io/fjw/fjw.rss"/>
                <style>
                    footer {font-size: x-small;}
                    article {margin-bottom: 4em;}
                    a {color: red; }
                </style>
            </head>
            <body>
                <h1>Post von Wagner</h1>
                <xsl:for-each select="rss/channel/item">
                    <article>
                        <header>
                            <h2>
                                <a target="_blank">
                                    <xsl:attribute name="href">
                                        <xsl:value-of select="link"/>
                                    </xsl:attribute>
                                    <xsl:value-of select="title"/>
                                </a>
                            </h2>
                        </header>
                        <section>
                            <xsl:value-of select="content:encoded" disable-output-escaping="yes"/>
                        </section>
                        <footer>
                            <xsl:value-of select="pubDate"/>
                        </footer>
                    </article>
                </xsl:for-each>
            </body>
        </html>
    </xsl:template>
</xsl:stylesheet>
