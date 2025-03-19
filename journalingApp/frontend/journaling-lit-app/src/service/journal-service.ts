import {
    PageResponse,
    PageWithContent
} from "../abstract/JournalTypes"

class JournalError extends Error { }

export class JournalService {
    constructor (private baseURL: string) { }

    public async getPage(id: string): Promise<PageResponse | undefined> {
        return this.fetchPage(id, false)
    }

    public async getPageWithContent(id: string): Promise<PageWithContent | undefined> {
        return this.fetchPage(id, true)
    }

    public async getPageList(): Promise<PageResponse[] | undefined> {
        return this.fetchPageList()
    }

    public async createPage(page: PageWithContent): Promise<void | object> {
        const response = await fetch(this.baseURL + "/pages", {
            method: "POST",
            body: JSON.stringify(page)
        });

        return this.upgradeErrors(response, "There was an issue creating a page")
    }

    public async updatePage(id: string, page: PageWithContent): Promise<void | object> {
        const response = await fetch(this.baseURL + `/pages/${id}`, {
            method: 'PUT',
            body: JSON.stringify(page)
        });

        return this.upgradeErrors(response, "There was an issue updating the page")
    }

    public async deletePage(id: string) {
        const response = await fetch(this.baseURL + `/pages/${id}`, {
            method: 'DELETE'
        });

        return this.upgradeErrors(response, "There was an issue deleting the page")
    }

    private async fetchPage<T>(id: string, withContent: boolean): Promise<T | undefined> {
        const endpoint = withContent ? `/pages/${id}/content` : `/pages/${id}`
        try {
            const response = await fetch(this.baseURL + endpoint, {
                method: 'GET'
            });
            if (response.ok) {
                const page = await response.json();
                return page;
            } else {
                return undefined
            }
        } catch (error) {
            return undefined
        }
    }

    private async fetchPageList(): Promise<PageResponse[] | undefined> {
        try {
            const response = await fetch(this.baseURL + "/pages", {
                method: 'GET'
            });
            if (response.ok) {
                const page = await response.json();
                return page;
            } else {
                return undefined
            }
        } catch (error) {
            return undefined
        }
    }

    private async upgradeErrors(response: Response, baseMessage: string) {
        try {
            if (response.ok) {
                return await response.json()
            } else {
                const data = await response.json();
                const errorMessage = data.error ? baseMessage + ":" + data.error : baseMessage;
                throw new JournalError(errorMessage)
            }
        } catch (error) {
            if (error instanceof JournalError) {
                throw error
            }

            throw new JournalError(baseMessage)
        }
    }
    
}