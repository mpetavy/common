console.info("test");
const names = ["a","b","c","d","e"];

let x = 3/0;

console.table(names);

const person = {
    name: "John Doe",
    street: "123 Main St",
    city: "Springfield",
    state: "IL",
    zip: "62704",
    country: "USA"
};

console.table(person);

const map = new Map();
map.set(1, 'Jack');
map.set(2, 'Jill');
map.set('animal', 'Elephant');

console.table(map);

const name = 'Marcel';

console.log(`Hello ${name}`);

console.error("This is an error");

throw new Error("This is a thrown error");
