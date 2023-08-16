const defaultImg = "https://og.atlasgo.io/image?title=Open-source%20database%20schema%20management"

// getImage returns the share image URL for a page/blog post. Order of evaluation is:
// the `image` attribute in the document front-matter, the `shareText` attribute in the
// document `front-matter`, the document's title, and finally the default image.
export function getImage(metadata) {
    const {frontMatter, title} = metadata
    if (frontMatter.image) {
        return frontMatter.image
    }
    if (frontMatter.shareText) {
        return `https://og.atlasgo.io/image?title=${encodeURIComponent(frontMatter.shareText)}`
    }
    if (title) {
        return `https://og.atlasgo.io/image?title=${encodeURIComponent(title)}`
    }
    return defaultImg
}