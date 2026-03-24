declare module "marked-terminal" {
  interface Options {
    width?: number;
    reflowText?: boolean;
    showSectionPrefix?: boolean;
    tab?: number;
    unescape?: boolean;
    emoji?: boolean;
    code?: any;
    blockquote?: any;
    heading?: any;
    firstHeading?: any;
    strong?: any;
    em?: any;
    codespan?: any;
    del?: any;
    link?: any;
    href?: any;
    listitem?: any;
    text?: any;
    tableOptions?: any;
  }
  export function markedTerminal(options?: Options): any;
  export default markedTerminal;
}
