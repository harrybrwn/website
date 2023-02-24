export interface Bookmark {
  url: string;
  name: string;
  description: string;
  tags: string[];
}

export interface Bookmarks {
  links: Bookmark[];
}

/**
 * bookmarks will fetch a list of bookmarks from the api.
 * @returns list of bookmarks
 */
export const bookmarks = async (): Promise<Bookmarks> => {
  return fetch(`${window.location.origin}/api/bookmarks`).then(
    (resp: Response) => {
      if (!resp.ok) {
        throw new Error("could not get bookmarks");
      }
      return resp.json();
    }
  );
};
