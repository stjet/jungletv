export const BRIDGE_VERSION = 3;
// methods the child can call on the parent
export type ParentMethods = {
    bridgeVersion: () => number;
    applicationID: () => string;
    serverMethod: (method: string, ...args: any[]) => Promise<any>;
    navigateToApplicationPage: (pageID: string, applicationID?: string) => void;
    navigate: (to: string) => void;
    alert: (message: string, title: string, buttonLabel: string) => Promise<void>;
    confirm: (question: string, title: string, positiveAnswerLabel: string, negativeAnswerLabel: string) => Promise<boolean>;
    prompt: (question: string, title: string, placeholder: string, initialValue: string, positiveAnswerLabel: string, negativeAnswerLabel: string) => Promise<string>;
    userAddress: () => Promise<string>;
    userPermissionLevel: () => Promise<string>;
}

// events that the parent can trigger on the child
export type ParentEvents = {
    "mounted": MountEventArgs,
    "destroyed": undefined,
    "connected": undefined,
    "disconnected": undefined,
    "eventForClient": ApplicationEventArgs,
    "themeChanged": ThemeChangedEventArgs,
}

// methods the parent can call on the child
export type ChildMethods = {}

// events that the child can trigger on the parent
export type ChildEvents = {
    "handshook": undefined,
    "eventForServer": ApplicationEventArgs,
    "pageTitleUpdated": string,
}

export type MountEventArgs = {
    role: "standalone" | "activity",
    applicationID: string,
    applicationVersion: Date,
    pageID: string,
}

export type ApplicationEventArgs = {
    name: string,
    args: any[],
}

export type ThemeChangedEventArgs = {
    darkMode: boolean,
}