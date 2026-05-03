import React from "react";

const IndexReact = () => {
  return (
    <div className="relative flex min-h-screen flex-col justify-center overflow-hidden bg-gray-100 text-gray-600 dark:text-gray-400 dark:bg-gray-900 py-6 sm:py-12">
      <div className="relative bg-white dark:bg-gray-800 px-6 pt-10 pb-8 shadow-xl ring-1 ring-gray-900/5 sm:mx-auto sm:max-w-xl sm:rounded-lg sm:px-10">
        <div className="mx-auto">
          <div className="flex items-center justify-center space-x-6">
            <img
              src="https://avatars.githubusercontent.com/u/109903896?s=200&v=4"
              alt="Lemmego"
              className="w-20 h-20"
            />
          </div>
          <div className="divide-y divide-gray-300 dark:divide-gray-700">
            <div className="py-8 text-base leading-7">
              <p>
                <strong>Lemmego</strong> is a modern, full-stack web development
                framework built with Go.
              </p>
              <hr className="my-5" />
              <p>
                <a
                  href="https://lemmego.github.io"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-semibold text-red-500 hover:text-red-600"
                >
                  Read the docs
                </a>{" "}
                to learn more about Lemmego.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default IndexReact;
