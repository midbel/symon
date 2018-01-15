const host = "http://localhost:9090"
const api = {
  process: `${host}/process/`,
}

const criteria = {
  user: "",
  group: "",
  command: "",
  state: "",
};

const app = new Vue({
  el: "#symon",
  data: {
    process: [],
    error: "",
    interval: undefined,
    rate: 1000,
    criteria: criteria,
  },
  mounted() {
    this.interval = setInterval(_.bind(this.update, this), this.rate);
    this.update();
  },
  computed: {
    all() {
      let keys = [];
      if ( this.criteria.command.length >= 3 ) {
        key.push(d => d.command.indexOf(this.criteria.command) >= 0)
      }
      if (!keys.length) {
        return this.process;
      }
      console.log(_.filter(this.process, keys))
      return _.filter(this.process, keys);
    },
    users() {
      return this.extract("user");
    },
    groups() {
      return this.extract("group");
    },
    commands() {
      return this.extract("process");
    },
    states() {
      return this.extract("state");
    },
  },
  methods: {
    extract(a) {
      return _.chain(this.process).map(p => p[a]).uniq().sort().value();
    },
    update() {
      fetch(api.process, {headers: {accept: "application/json"}}).then(r => {
        if (!r.ok) {
          return Promise.reject(r.statusText());
        }
        return r.json();
      }).then(rs => {
        this.process = _.sortBy(rs, [d => d.ppid, d => d.pid]);
        this.error = "";
      }).catch(err => {
        this.error = err;
        if (this.interval) {
          clearInterval(this.interval);
        }
      });
    },
  },
});
