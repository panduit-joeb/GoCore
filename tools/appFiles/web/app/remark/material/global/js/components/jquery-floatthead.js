/*!
 * remark (http://getbootstrapadmin.com/remark)
 * Copyright 2017 amazingsurge
 * Licensed under the Themeforest Standard Licenses
 */
$.components.register("floatThead", {
  mode: "default",
  defaults: {
    top: function() {
      return $('.site-navbar').outerHeight();
    },
    position: 'absolute'
  }
});
