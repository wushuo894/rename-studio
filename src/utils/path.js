/**
 * 按 macOS、Linux 和 Windows 的路径分隔符拆分路径。
 * 这里不依赖浏览器 URL，避免空格和中文路径被错误编码。
 */
export function splitPath(filePath) {
    const separatorIndex = Math.max(filePath.lastIndexOf('/'), filePath.lastIndexOf('\\'))
    return {
        directory: separatorIndex >= 0 ? filePath.slice(0, separatorIndex) : '',
        name: separatorIndex >= 0 ? filePath.slice(separatorIndex + 1) : filePath,
        separator: filePath.includes('\\') ? '\\' : '/',
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

/** 使用原路径的分隔符拼接目录与新文件名。 */
export function joinSiblingPath(sourcePath, newName) {
    const {directory, separator} = splitPath(sourcePath)
    return directory ? `${directory}${separator}${newName}` : newName
}
