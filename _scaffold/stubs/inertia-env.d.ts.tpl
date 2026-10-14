import "@inertiajs/core";

// Inertia's own types, told what this framework actually puts on the wire.
//
// `errorValueType` defaults to `string`, so `errors.email` types as a string
// and `errors.email.join(", ")` does not compile. Lemmego flashes
// `shared.ValidationErrors`, which is a `map[string][]string` — see
// bootstrap/errmap.go — so a field can carry several messages and the array is
// the truth. Without this declaration the generated forms were correct at
// runtime and rejected by `tsc`, which is the worst way round: the code works,
// so nobody believes the type error, and the typecheck gets skipped instead of
// the types getting fixed.
//
// `sharedPageProps` is where props every page receives belong. Adding them
// here is what turns `usePage().props.foo` from `unknown` into a real type,
// because `PageProps` is an open `[key: string]: unknown` index signature and
// cannot narrow on its own.
declare module "@inertiajs/core" {
  export interface InertiaConfig {
    errorValueType: string[];
    sharedPageProps: {
      // Flashed by bootstrap/errmap.go and by res.Flash. Present on every
      // page, absent most of the time, which is why it is optional rather
      // than a string that is usually empty.
      message?: string;
      // Add the rest of this application's shared props here — whatever
      // bootstrap registers with inertia.Share — so every page reads them
      // with a type instead of a cast.
    };
  }
}
