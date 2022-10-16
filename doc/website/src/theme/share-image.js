const defaultImg = "https://atlas-og-img.vercel.app/**Atlas%20%7C**%20Open-source%20database%20schema%20management.png?theme=dark"

// getImage returns the share image URL for a page/blog post. Order of evaluation is:
// the `image` attribute in the document front-matter, the `shareText` attribute in the
// document `front-matter`, the document's title, and finally the default image.
export function getImage(metadata) {
    const {frontMatter, title} = metadata
    if (frontMatter.image) {
        return frontMatter.image
    }
    if (frontMatter.shareText) {
        return `https://atlas-og-img.vercel.app/${encodeURIComponent(frontMatter.shareText)}.png?theme=dark`
    }
    if (title) {
        return `https://atlas-og-img.vercel.app/${encodeURIComponent(title)}.png?theme=dark`
    }
    return defaultImg
}