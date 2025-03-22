import { LitElement, html } from 'lit'
import { customElement, state } from 'lit/decorators.js'
import { provide } from "@lit/context"
import { PageResponse } from "../abstract/JournalTypes"
import { appContext, AppContext } from '../data/context'
import { JournalService } from '../service/journal-service'


@customElement("journ-page-list")
export class JournPageList extends LitElement {

    @provide({ context: appContext })   
    context: AppContext = { journalService: new JournalService("http://localhost:8080") };

    @state()
    private pageList?: PageResponse[];

    protected async willUpdate() {
        if (this.pageList === undefined && this.context) {
            console.log("Fetching page list...");
            this.pageList = await this.context.journalService.getPageList();
        }
    }

    render() {
        if (!this.pageList) {
            return html`<p>Loading...</p>`; // Show a loading state if data isn't ready
        }
        return html `
            <div>
                <table>
                    <thead>
                        <tr>
                            <th>Title</th>
                            <th>ID</th> 
                        </tr>
                    </thead>
                    <tbody>
                        ${this.pageList?.map(p => this.renderPage(p))}
                    </tbody>
                </table>
            </div>
        `
    }

    renderPage(page: PageResponse) {
        return html`
            <tr>
                <td>${page.title}</td>
                <td>${page.id}</td>
            </tr>
        `
    }
}
