/** 只有 Windows 将反斜线视为路径分隔符。 */
export function splitPath(filePath, windows) {
    const slash = filePath.lastIndexOf('/')
    const separator = windows ? Math.max(slash, filePath.lastIndexOf('\\')) : slash
    return {
        directory: separator >= 0 ? filePath.slice(0, separator) : '',
        name: filePath.slice(separator + 1),
    }
}

/** 将文件名拆分成基础名称和扩展名，隐藏文件不会被误判为只有扩展名。 */
export function splitFileName(fileName) {
    const lastDot = fileName.lastIndexOf('.')
    if (lastDot <= 0) return {baseName: fileName, extension: ''}
    return {
        baseName: fileName.slice(0, lastDot),
        extension: fileName.slice(lastDot + 1),
    }
}
