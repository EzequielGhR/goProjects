import { LitElement, html, css } from 'lit'
import { customElement, state } from 'lit/decorators.js'
import { PageResponse, PageWithContent } from "../abstract/JournalTypes"
import { JournalService } from '../service/journal-service'


@customElement("journ-page-list")
export class JournPageList extends LitElement {

    static styles = css`
        :host {
            display: flex;
            height: 100vh;
            width: 100vw;
            font-family: Arial, sans-serif;
        }

        .container {
            display: flex;
            width: 100%;
            border: 1px solid #ddd;
        }

        /* Sidebar Styles */
        .sidebar {
            width: 30%;
            max-width: 250px;
            background: #333;  /* Dark sidebar */
            color: white;  /* Better contrast */
            overflow-y: auto;
            border-right: 1px solid #ddd;
            padding: 8px;
        }

        .sidebar h2 {
            text-align: center;
            margin-bottom: 10px;
            font-size: 1.2rem;
        }

        .page-item {
            padding: 12px;
            border-bottom: 1px solid #555;
            cursor: pointer;
            transition: background 0.2s, transform 0.1s;
        }

        .page-item:hover {
            background: #444;
            transform: scale(1.02);
        }

        .selected {
            background: #007bff;
            color: white;
            font-weight: bold;
            box-shadow: inset 0 0 5px rgba(0, 0, 0, 0.2);
        }

        /* Content Area */
        .content {
            flex: 1;
            padding: 16px;
            background: #f9f9f9;  /* Light gray background */
            color: #222;  /* Darker text for contrast */
            overflow-y: auto;
        }

        .content h2 {
            color: #333;
        }

        .content p {
            line-height: 1.5;
        }

        .loading {
            text-align: center;
            font-style: italic;
            color: gray;
        }
    `;


    protected service = JournalService.newLocal();

    @state()
    private pageList?: PageResponse[];

    @state()
    private selectedPage?: PageWithContent;

    @state()
    private isLoadingContent = false;

    protected async willUpdate() {
        if (this.pageList === undefined) {
            console.log("Fetching page list...");
            this.pageList = await this.service.getPageList();
        }
    }

    private async selectPage(pageId: string) {
        if (this.selectedPage?.id === pageId) {
            return;
        }

        console.log(`Selecting page with id "${pageId}" ...`)
        this.isLoadingContent = true;
        this.selectedPage = await this.service.getPageWithContent(pageId);
        this.isLoadingContent = false;
    }

    render() {
        if (!this.pageList) {
            return html`<p>Loading...</p>`; // Show a loading state if data isn't ready
        }
        return html`
            <div class="container">
                <!-- Sidebar -->
                <div class="sidebar">
                    <h2>Pages</h2>
                    ${this.pageList.map(page => this.renderPageTitle(page))}
                </div>
                <!-- Details View -->
                <div class="content">
                    ${this.renderContent()}
                </div>
            </div>
        `
    }

    renderPageTitle(page: PageResponse | PageWithContent) {
        const isSelected = this.selectedPage?.id === page.id;

        return html`
            <div class="page-item ${isSelected ? 'selected' : ''}" @click=${() => this.selectPage(page.id)}>
                ${page.title}
            </div>
        `
    }

    renderContent() {
        if (this.isLoadingContent) {
            return html`<p class="loading">Loading page content...</p>`;
        }

        if (!this.selectedPage) {
            return html`<p>Select a page to view details</p>`
        }

        console.log(this.selectedPage)
        return html`
            <h2>${this.selectedPage.title}</h2>
            <p>ID: ${this.selectedPage.id}</p>
            <p>${this.selectedPage.content}</p>
        `
    }
}
