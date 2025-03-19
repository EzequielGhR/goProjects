export type Page = {
    id: string;
    title: string;
    path: string;
}

export type PageWithContent = Page & {
    content: string
}

export type PageResponse = Page & {
    elapsed: Number
}
