import { Component, } from "react";
import type { ErrorInfo, ReactNode, } from "react";
import { ErrorPage, } from "@/pages/error";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props,) {
    super(props,);
    this.state = { error: null, };
  }

  static getDerivedStateFromError(error: Error,) {
    return { error, };
  }

  componentDidCatch(error: Error, info: ErrorInfo,) {
    console.error("[ErrorBoundary]", error, info.componentStack,);
  }

  render() {
    if (this.state.error) {
      return <ErrorPage error={this.state.error} reset={() => this.setState({ error: null, },)} />;
    }
    return this.props.children;
  }
}
