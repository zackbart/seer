declare module "marked-terminal" {
  interface Options {
    width?: number;
    reflowText?: boolean;
    showSectionPrefix?: boolean;
    tab?: number;
  }
  export function markedTerminal(options?: Options): any;
  export default markedTerminal;
}
