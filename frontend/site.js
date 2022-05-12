"use strict";

const twitter = {
  site: "@harryb998",
  creator: "@harryb998",
  card: "summary_larg_image",
};

const domain = "harrybrwn.com";
const built = new Date();

const common = {
  author: "Harry Brown",
  domain,
  description: "The home page of a humble backend software engineer.",
  previewImage: "https://harrybrwn.com/static/img/goofy.jpg",
  github: "https://github.com/harrybrwn",
  linkedin: "https://www.linkedin.com/in/harrison-brown-88823b185/",
  built,
  og: true,
  robots: "index, archive, follow",
  twitter,
};

module.exports = {
  domain: common.domain,
  pages: {
    index: {
      title: "Harry Brown",
      ...common,
    },
    remora: {
      title: "Remora | Harry Brown",
      ...common,
    },
    blog: {
      title: "Blog | Harry Brown",
      ...common,
    },
    admin: {
      title: "Admin Panel",
      ...common,
    },
    chatroom: {
      title: "Harry's webchat",
      ...common,
    },
    games: {
      title: "Games",
      ...common,
    },
    harry_y_tanya: {
      title: "Tanya y Harry",
      ...common,
    },
    tanya: {
      title: "Tanya Rivera",
      domain: common.domain,
      description: "Tanya Rivera's website.",
      previewImage: "https://harrybrwn.com/static/img/tanya/cities.jpg",
      linkedin: "https://www.linkedin.com/in/",
      built: new Date(),
      twitter: {
        site: "@Tanya_riv",
        creator: "@Tanya_riv",
        card: "summary_large_image",
      },
      og: {
        type: "website",
      },
    },
    404: {
      title: "404 Not Found",
      domain: "harrybrwn.com",
    },
    "50x": {
      title: "Error",
      ...common,
    },
    invite: {
      title: "You're Invited!",
      ...common,
      robots: false,
    },
    invite_email: {
      title: "You're Invited!",
      ...common,
      robots: false,
    },
  },
};
