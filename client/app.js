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
      let chain = _.chain(this.process);
      if ( this.criteria.command.length >= 3 ) {
        chain = chain.filter(d => d.command.indexOf(this.criteria.command) >= 0);
      }
      chain = chain.filter(d => this.criteria.user == "" || d.user == this.criteria.user);
      chain = chain.filter(d => this.criteria.group == "" || d.group == this.criteria.group);

      return chain.value()
    },
    users() {
      return [""].concat(this.extract("user"));
    },
    groups() {
      return [""].concat(this.extract("group"));
    },
    commands() {
      return [""].concat(this.extract("process"));
    },
    states() {
      return [""].concat(this.extract("state"));
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
