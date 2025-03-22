import { createContext } from "@lit/context";
import { JournalService } from "../service/journal-service"

export type AppContext = {
    journalService: JournalService
}

export const appContext = createContext<AppContext>(Symbol("app-context"))